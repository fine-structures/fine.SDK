package lib2x3

import (
	"fmt"
	"io"
	"log"
	"sort"
	"sync"

	"github.com/fine-structures/fine-sdk-go/go2x3"
	"github.com/fine-structures/fine-sdk-go/lib2x3/graph"
	walker "github.com/fine-structures/fine-sdk-go/lib2x3/graph-walker"
)

func NewGraph(Xsrc *Graph) *Graph {
	X := graphPool.Get().(*Graph)
	X.Init(Xsrc)
	return X
}

func NewGraphFromDef(graphDef []byte) (*Graph, error) {
	X := graphPool.Get().(*Graph)
	err := X.InitFromDef(graphDef)
	if err != nil {
		return nil, err
	}
	return X, nil
}

// VtxList is an ordered sequence of VtxTypes
type VtxList []VtxType

func (V VtxList) Len() int           { return len(V) }
func (V VtxList) Less(i, j int) bool { return V[i] < V[j] }
func (V VtxList) Swap(i, j int)      { V[i], V[j] = V[j], V[i] }

// GraphEncoding a fully serialized Graph. See initFromEncoding() for format info.
type GraphEncoding []byte

func (Xenc GraphEncoding) GraphInfo() go2x3.GraphInfo {
	info := go2x3.GraphInfo{
		NumParticles: Xenc[0],
		NumVertex:    Xenc[1],
		NegEdges:     Xenc[2],
		NegLoops:     Xenc[3],
		PosLoops:     Xenc[4],
	}
	info.PosEdges = info.NumEdges() - info.NegEdges
	return info
}

// (VtxID << 2) | (slotsRemaining)
type VtxEdgeSlots uint16

func NewSlotsForVtxType(vtxID VtxID, vtxType VtxType) VtxEdgeSlots {
	return VtxEdgeSlots(vtxID<<2) | VtxEdgeSlots(vtxType.NumEdges())
}

func (slot VtxEdgeSlots) NumOpenEdgeSlots() VtxCount {
	return VtxCount(slot) & 0x3
}

func (slot VtxEdgeSlots) VtxID() VtxID {
	return VtxID(slot >> 2)
}

func (slot VtxEdgeSlots) UseSlot(numEdges int32) (VtxEdgeSlots, bool) {
	if int32(slot&0x3) >= numEdges {
		return slot - VtxEdgeSlots(numEdges), true
	}
	return slot, false
}

type OpenEdgeSlots []VtxEdgeSlots

func (slotsSet OpenEdgeSlots) CountOpenEdgeSlots() VtxCount {
	count := VtxCount(0)
	for _, slot := range slotsSet {
		count += slot.NumOpenEdgeSlots()
	}
	return count
}

const (
	CmdAddEdge EncoderCmd = 1 << 14
	CmdAddVtx  EncoderCmd = 2 << 14
)

type EncoderCmd uint16

func NewAddVtxCmd(vtxType VtxType) EncoderCmd {
	return CmdAddVtx | EncoderCmd(vtxType)
}

func NewAddEdgeCmd(Va, Vb VtxID) EncoderCmd {
	if Va < Vb {
		return CmdAddEdge | (EncoderCmd(Va) << 7) | EncoderCmd(Vb)
	} else {
		return CmdAddEdge | (EncoderCmd(Vb) << 7) | EncoderCmd(Va)
	}
}

func (cmd EncoderCmd) IsAddVtxCmd() VtxType {
	if cmd&CmdAddVtx != 0 {
		return VtxType(cmd) & VtxTypeMask
	}
	return V_nil
}

func (cmd EncoderCmd) IsAddEdgeCmd() (isAddEdge bool, Va, Vb VtxID) {
	if cmd&CmdAddEdge != 0 {
		return true, VtxID(cmd>>7) & VtxIDMask, VtxID(cmd) & VtxIDMask
	}
	return false, 0, 0
}

type Graph struct {
	partCount int               // number of particles (i.e. matrix partitions).  Zero if not yet calculated
	vtxCount  int               // number of assigned Vtx in []vtx
	edgeCount int               // number of assigned EdgeIDs in []edges
	vtx       [MaxVtxID]VtxType // poles assignment
	edges     [MaxEdges]EdgeID  // edges assignment
	dirty     bool
	xstate    graphState
	vm        graph.VtxGraphVM
	Def       graph.GraphDef
}

func (X *Graph) MakeCopy() go2x3.State {
	return NewGraph(X)
}

func (X *Graph) Edges() EdgeList {
	return X.edges[:X.edgeCount]
}

func (X *Graph) Vtx() []VtxType {
	return X.vtx[:X.vtxCount]
}

func (X *Graph) Len() int           { return X.vtxCount }
func (X *Graph) Less(i, j int) bool { return X.vtx[i] < X.vtx[j] }
func (X *Graph) Swap(i, j int) {
	X.vtx[i], X.vtx[j] = X.vtx[j], X.vtx[i]
	Vi := VtxID(i + 1)
	Vj := VtxID(j + 1)

	// For the VtxIDs being swapped, also swap their corresponding edge connections
	X.Edges().SwapVtxID(Vi, Vj)
}

// Returns the number of particles (partitions) in this graph
func (X *Graph) NumParticles() byte {
	if X.partCount > 0 {
		return byte(X.partCount)
	}

	// We find number of total partitions.  Start by assuming each vertex its own partition.
	// Each time we connect two vertices with an edge, propagate their connectedness.
	var vtxBuf [MaxVtxID]VtxID
	Nv := VtxID(X.vtxCount)
	vtx := vtxBuf[:Nv]
	for i := VtxID(0); i < Nv; i++ {
		vtx[i] = i + 1
	}
	for _, edge := range X.Edges() {
		va, vb := edge.VtxIdx()
		v_lo := vtx[va]
		v_hi := vtx[vb]
		if v_lo == v_hi {
			continue
		}
		if v_lo > v_hi {
			v_lo, v_hi = v_hi, v_lo
		}
		for i, vi := range vtx {
			if vi == v_hi {
				vtx[i] = v_lo
			}
		}
	}

	// The number of unique values in the vtx list is the number of partitions
	pcount := 0
	if Nv > 0 {
		pcount++
	}
	for _, vi := range vtx {
		newPart := true
		for j := 0; j < pcount; j++ {
			if vtx[j] == vi {
				newPart = false
			}
		}
		if newPart {
			vtx[pcount] = vi
			pcount++
		}
	}

	X.partCount = pcount
	return byte(X.partCount)
}

func (X *Graph) VertexCount() int {
	return X.vtxCount
}

func (X *Graph) CountLoops() (negLoops, posLoops byte) {
	for _, vi := range X.Vtx() {
		negLoops += vi.NegLoops()
		posLoops += vi.PosLoops()
	}
	return
}

func (X *Graph) CountEdges() (totalPos, totalNeg byte) {
	for _, edge := range X.Edges() {
		numPos, numNeg := edge.EdgeType().NumPosNeg()
		totalPos += numPos
		totalNeg += numNeg
	}
	return
}

func (X *Graph) GraphInfo() go2x3.GraphInfo {
	negLoops, posLoops := X.CountLoops()
	posEdges, negEdges := X.CountEdges()

	return go2x3.GraphInfo{
		NumParticles: X.NumParticles(),
		NumVertex:    byte(X.VertexCount()),
		NegEdges:     negEdges,
		PosEdges:     posEdges,
		NegLoops:     negLoops,
		PosLoops:     posLoops,
	}
}

func (X *Graph) Init(Xsrc *Graph) {
	if X == Xsrc {
		return
	}

	X.onGraphChanged()

	if Xsrc == nil {
		X.vtxCount = 0
		X.edgeCount = 0
		X.Def.AssignFrom(nil)
		return
	}
	X.partCount = Xsrc.partCount
	X.vtxCount = Xsrc.vtxCount
	X.edgeCount = Xsrc.edgeCount
	X.Def.AssignFrom(&Xsrc.Def)
	copy(X.Vtx(), Xsrc.Vtx())
	copy(X.Edges(), Xsrc.Edges())
}

func (X *Graph) AssignFromCmds(cmds []EncoderCmd) {
	X.Init(nil)

	for _, cmd := range cmds {
		if vtxType := cmd.IsAddVtxCmd(); vtxType != V_nil {
			X.vtx[X.vtxCount] = vtxType
			X.vtxCount++
		} else if isAddEdge, Va, Vb := cmd.IsAddEdgeCmd(); isAddEdge {
			X.edges[X.edgeCount] = PosEdge.FormEdge(Va, Vb)
			X.edgeCount++
		}
	}

	X.combineMultiEdges()
}

func (X *Graph) InitFromDef(graphDef []byte) error {
	X.Def.AssignFrom(nil)
	err := X.Def.Unmarshal(graphDef)
	if err != nil {
		return err
	}
	err = X.initFromEncoding(X.Def.GraphEncoding)
	if err != nil {
		return err
	}
	return nil
}

// combineMultiEdges combines multiple edges connecting the same vertices into the proper EdgeType
func (X *Graph) combineMultiEdges() {

	// Sort by Va and Vb only (so we can find edges for the same two vtx).
	// We assume that each edge is already canonic in that Va < Vb
	edges := X.Edges()
	sort.Slice(edges, func(i, j int) bool {
		ab_i := edges[i] &^ EdgeTypeMask
		ab_j := edges[j] &^ EdgeTypeMask
		return ab_i < ab_j
	})

	Ne := len(edges)
	D := 0
	for L := 0; L < Ne; D++ {
		edge_L := edges[L]
		ab_L := edge_L &^ EdgeTypeMask
		et_L := edge_L.EdgeType()

		// Find the end of this edge run (usually a run is only a single edge)
		R := L + 1
		for ; R < Ne; R++ {
			ab_R := edges[R] &^ EdgeTypeMask
			if ab_L != ab_R {
				break
			}
			et_R := edges[R].EdgeType()
			et_L = et_L.CombineWith(et_R)
		}
		if R-L > 1 {
			edges[D] = edge_L.ChangeEdgeType(et_L)
		} else if D != L {
			edges[D] = edge_L
		}
		L = R
	}
	if D != Ne {
		X.edgeCount = D
	}
}

var (
	quote   = []byte("\"")
	space   = []byte(" ")
	comma   = []byte(",")
	newline = []byte("\n")
)

func (X *Graph) Canonize(normalize bool) error {
	X.Traces(0) // Make sure graph is flushed to X.xstate
	X.xstate.Canonize()
	return nil
}

func (X *Graph) WriteAsString(out io.Writer, opts go2x3.PrintOpts) {
	X.Canonize(false) // TODO: remove this when we can print output for any case: 1) un-canonized, 2) canonized, 3) canonized+normalized

	var scrap [512]byte
	encFull, _ := X.MarshalOut(scrap[:0], go2x3.AsAscii)
	fmt.Fprintf(out, "p=%d,v=%d,%q,%q,", X.NumParticles(), X.VertexCount(), encFull, "")

	if opts.Graph {
		X.WriteAsGraphExprStr(out)
	}
	if opts.Matrix {
		X.WriteAsMatrixStr(out)
	}
	if opts.NumTraces != 0 {
		X.WriteTracesAsCSV(out, opts.NumTraces)
	}

	//out.Write(newline)

	if opts.CycleSpec {
		out.Write(newline)
		X.vm.Canonize()
		X.vm.PrintCycleSpectrum(12, out)
	}

}

func (X *Graph) WriteTracesAsCSV(out io.Writer, numTraces int) {
	TX := X.Traces(numTraces)

	var buf [24]byte

	for _, TXi := range TX {
		out.Write(graph.PrintInt(buf[:], TXi))
		out.Write(comma)
	}
}

func (X *Graph) WriteAsGraphExprStr(out io.Writer) {
	var buf [MaxVtxID]byte
	negLoops := buf[:X.vtxCount]

	for i := range negLoops {
		negLoops[i] = X.vtx[i].NegLoops()
	}

	printVtx := func(vi VtxID) {
		var buf [8]byte
		s := graph.PrintInt(buf[:4], int64(vi))
		neg := negLoops[vi-1]

		if neg > 0 {
			negLoops[vi-1] = 0
			for i := byte(0); i < neg; i++ {
				s = append(s, '^')
			}
		}
		out.Write(s)
	}

	out.Write(quote)

	// Write out single verts
	needsBreak := false
	for vi, v := range X.Vtx() {
		if v.NumEdges() == 0 {
			if needsBreak {
				out.Write(space)
			}
			printVtx(VtxID(vi + 1))
			needsBreak = true
		}
	}

	// Print edges -- combine vtx where possible
	{
		Ne := X.edgeCount
		e := make([]EdgeID, Ne)
		for i, edge := range X.Edges() {
			e[i] = edge
		}

		var b_prev VtxID
		for i := 0; i < Ne; i++ {

			// Look for an edge we can combine
			edge := e[i]
			if b_prev != 0 {
				for j := i; j < Ne; j++ {
					aj, bj := e[j].VtxAB()
					if aj == b_prev || bj == b_prev {
						edge = e[j]
						e[j] = e[i]
						break
					}
				}
			}

			a, b := edge.VtxAB()
			if b == b_prev {
				a, b = b, a
			}

			// If we can't combine, print a sep then first vtx
			if a != b_prev {
				if needsBreak {
					out.Write(space)
				}
				printVtx(a)
			}
			fmt.Fprint(out, edge.EdgeType().String())
			printVtx(b)
			b_prev = b
			needsBreak = true
		}
	}
	out.Write(quote)
	out.Write(comma)
}

func (X *Graph) WriteAsMatrixStr(out io.Writer) {
	Nv := X.VertexCount()

	var buf [8]byte
	var Xm [16 * 16]int8

	// Set matrix diagonal values from vertices
	for i := 0; i < Nv; i++ {
		v := X.vtx[i]
		Xm[i+i*Nv] = v.NetLoops()
	}

	// Set edge values
	for _, ei := range X.Edges() {
		a, b := ei.VtxIdx()
		edges := int8(ei.EdgeSum())
		Xm[a+b*Nv] = edges
		Xm[b+a*Nv] = edges
	}

	out.Write([]byte("\"{"))
	for row := 0; row < Nv; row++ {
		if row > 0 {
			out.Write(comma)
		}
		out.Write([]byte("{"))
		for j := 0; j < Nv; j++ {
			Xij := int64(Xm[j+row*Nv])
			if j > 0 {
				out.Write(comma)
			}
			out.Write(graph.PrintInt(buf[:], Xij))
		}
		out.Write([]byte("}"))
	}
	out.Write([]byte("}\","))

}

func (X *Graph) Reclaim() {
	if X != nil {
		graphPool.Put(X)
	}
}

var graphPool = sync.Pool{
	New: func() interface{} {
		return new(Graph)
	},
}

// Assigns this Graph from the given encoding generated by AppendGraphEncoding()
//
// Format: (most significant info first allows useful LSM searching/sorting)
//
//	NumParticles
//	NumVerts
//	NumNegEdges
//	NumNegLoops
//	NumLoops
//	<1..NumVerts>
//	    byte(VtxType)
//	NumEdgeIDs
//	<1..NumEdgeIDs>
//	    uint16(EdgeID) (edge sign(s), Va, Vb)
func (X *Graph) initFromEncoding(Xe GraphEncoding) error {
	X.Init(nil)

	info := Xe.GraphInfo()
	X.vtxCount = int(info.NumVertex)

	if int(info.NumParticles) > X.vtxCount {
		return go2x3.ErrBadEncoding
	}

	idx := int32(5)

	// read VtxTypes
	for i := 0; i < X.vtxCount; i++ {
		v := VtxType(Xe[idx])
		if v <= 0 || v > V_𝛾 {
			return go2x3.ErrBadEncoding
		}
		X.vtx[i] = v
		info.NegLoops -= v.NegLoops()
		info.PosLoops -= v.PosLoops()
		idx++
	}

	// consistency check
	if info.NegLoops != 0 || info.PosLoops != 0 {
		return go2x3.ErrBadEncoding
	}

	// Note this edge count is for edge *types*, so for example, PosPosEdge would be one edge.
	X.edgeCount = int(Xe[idx])
	idx++

	// read edges
	for i := 0; i < X.edgeCount; i++ {
		edge := (EdgeID(Xe[idx]) << 8) | EdgeID(Xe[idx+1])
		numPos, numNeg := edge.EdgeType().NumPosNeg()
		info.NegEdges -= numNeg
		info.PosEdges -= numPos
		X.edges[i] = edge
		idx += 2
	}

	// consistency check
	if info.NegEdges != 0 || info.PosEdges != 0 {
		return go2x3.ErrBadEncoding
	}

	return nil
}

func (X *Graph) appendGraphEncodingTo(buf []byte) []byte {
	info := X.GraphInfo()
	buf = info.AppendGraphEncodingHeader(buf)

	// Append VtxTypes
	for _, vi := range X.Vtx() {
		buf = append(buf, vi.Ord())
	}

	edges := X.Edges()

	// num edges + edges
	buf = append(buf, byte(len(edges)))
	for _, edge := range edges {
		buf = append(buf,
			byte(edge>>8),
			byte(edge))
	}

	return buf
}

// Concatenates Xsrc to the "end" of X
func (X *Graph) Concatenate(Xsrc *Graph) {
	v0 := VtxID(X.vtxCount)
	X.vtxCount += Xsrc.vtxCount
	for i, vtx := range Xsrc.Vtx() {
		X.vtx[v0+VtxID(i)] = vtx
	}

	e0 := X.edgeCount
	X.edgeCount += Xsrc.edgeCount
	for i, edge := range Xsrc.Edges() {
		a, b := edge.VtxAB()
		X.edges[e0+i] = edge.EdgeType().FormEdge(a+v0, b+v0)
	}

	X.onGraphChanged()
}

func (X *Graph) onGraphChanged() {

	// Reset generated info since the graph changed
	X.partCount = 0
	X.dirty = true
}

func (X *Graph) MarshalOut(out []byte, opts go2x3.MarshalOpts) ([]byte, error) {
	if opts&go2x3.AsGraphDef != 0 {
		X.Def.GraphEncoding = X.appendGraphEncodingTo(X.Def.GraphEncoding[:0])
		buf, err := X.Def.Marshal()
		if err != nil {
			return nil, err
		}
		return append(out, buf...), nil
	} else {
		return X.xstate.MarshalOut(out, opts)
	}
}

func ExportGraph(Xsrc *Graph, X *graph.VtxGraphVM) error {
	X.ResetGraph()
	Nv := Xsrc.VertexCount()
	if Xsrc == nil || Nv == 0 {
		return ErrNilGraph
	}

	// First, add edges that connect to the same vertex (loops)
	for i, vtyp := range Xsrc.Vtx() {
		vi := uint32(i + 1)
		numPos := int32(vtyp.PosLoops())
		numNeg := int32(vtyp.NegLoops())
		if err := X.AddEdge(numNeg, numPos, vi, vi); err != nil {
			panic(err)
		}
	}

	// Second, add edges connecting two different vertices
	for _, edge := range Xsrc.Edges() {
		ai, bi := edge.VtxAB()
		edgeType := edge.EdgeType()
		numPos, numNeg := edgeType.NumPosNeg()
		if err := X.AddEdge(int32(numNeg), int32(numPos), uint32(ai), uint32(bi)); err != nil {
			panic(err)
		}
	}

	return X.Validate()
}

// Traces returns a slice of the requested number of Traces.  If numTraces == 0, then the Traces length defaults to X.NumVerts()
// The slice should be considered immediate read-only.
func (X *Graph) Traces(numTraces int) go2x3.Traces {
	if X.dirty {
		if err := ExportGraph(X, &X.vm); err != nil {
			panic(err)
		}
		X.xstate.AssignGraph(X)
		X.dirty = false
	}

	return X.xstate.Traces(numTraces)
}

// PermuteVtxSigns emits a Graph for every possible vertex pole permutation of the given Graph.
//
// The callback handler should not make any changes to Xperm (with the exception of calling Traces())
func (X *Graph) PermuteVtxSigns(dst *go2x3.GraphStream) {

	Nv := X.VertexCount()
	if Nv == 0 {
		return
	}

	Xi := NewGraph(X)
	defer Xi.Reclaim()

	// Build the permutation we will traverse
	var span [MaxVtxID][4]VtxType
	permCount := int64(1)
	for vi := 0; vi < Nv; vi++ {
		vtxPerm := X.vtx[vi].VtxPerm()
		span[vi] = vtxPerm.Vtx
		permCount *= int64(vtxPerm.Num)
		Xi.vtx[vi] = span[vi][0]
	}

	for {
		dst.Outlet <- Xi.MakeCopy()
		permCount--

		// "Increment" to the next permutation
		carry := true
		for vi := 0; vi < Nv && carry; vi++ {
			v := Xi.vtx[vi]

			// If the vertex is a gamma, save work and just skip to the next digit since there are no pole permutations of a gamma vertex
			if v == V_𝛾 {
				continue
			}

			switch v {
			case span[vi][0]:
				v = span[vi][1]
			case span[vi][1]:
				v = span[vi][2]
			case span[vi][2]:
				v = span[vi][3]
			default:
				v = V_nil
			}

			// Is there a carry?
			if v == V_nil {
				v = span[vi][0]
			} else {
				carry = false
			}

			// Write the vertex change
			Xi.vtx[vi] = v
		}

		// Each time the graph changes, discard any calculated traces
		Xi.onGraphChanged()

		if carry {
			if permCount != 0 {
				panic("calculated number of VtxType permutations did not equal number of enumerations")
			}
			break
		}
	}

}

// PermuteEdgeSigns emits a Graph for every possible edge sign permutation of the given Graph.
//
// The callback handler should not make any changes to Xperm (with the exception of calling Traces())
func (X *Graph) PermuteEdgeSigns(dst *go2x3.GraphStream) {

	// If there's no edges to permute over, export only the given graph (which is just 0 or more single vertex particles).
	// Note that X.edgeCount is vertex pair count, so 2 or 3 edges of matching type will only show up as *one* element.
	Ne := X.edgeCount
	if Ne == 0 {
		dst.Outlet <- X.MakeCopy()
		return
	}

	Xi := NewGraph(X)
	defer Xi.Reclaim()

	// Build the permutation we will traverse
	permCount := int64(1)
	var span [MaxEdges][4]EdgeID
	for ei, edgeID := range Xi.Edges() {
		edgePerm := edgeID.EdgePerm()
		span[ei] = edgePerm.Edges
		permCount *= int64(edgePerm.Num)
		Xi.edges[ei] = span[ei][0]
	}

	for {
		dst.Outlet <- Xi.MakeCopy()
		permCount--

		// "Increment" to the next permutation
		carry := true
		for ei := 0; ei < Ne && carry; ei++ {
			e := Xi.edges[ei]

			switch e {
			case span[ei][0]:
				e = span[ei][1]
			case span[ei][1]:
				e = span[ei][2]
			case span[ei][2]:
				e = span[ei][3]
			default:
				e = 0
			}

			// Is there a carry?
			if e == 0 {
				e = span[ei][0]
			} else {
				carry = false
			}

			// Write the edge change
			Xi.edges[ei] = e
		}

		Xi.onGraphChanged()

		if carry {
			if permCount != 0 {
				panic("calculated number of VtxType permutations did not equal number of enumerations")
			}
			break
		}
	}
}

func EnumPureParticles(opts walker.EnumOpts) *go2x3.GraphStream {
	gw, err := NewGraphWalker()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		gw.EnumPureParticles(opts.VertexMax)
	}()

	return gw.EnumStream
}

// PBuilder
// PWalker
type GraphWalker struct {
	targetVtxCount VtxID
	vtxChoices     []VtxType
	openSlotsScrap [MaxVtxID]VtxEdgeSlots
	encoderScrap   [MaxVtxID * 4]EncoderCmd //  max cmds: max vertices + 3 edges per vertex
	EnumStream     *go2x3.GraphStream
}

func NewGraphWalker() (*GraphWalker, error) {

	gw := &GraphWalker{
		EnumStream: &go2x3.GraphStream{
			Outlet: make(chan go2x3.State, 1),
		},
		vtxChoices: []VtxType{V_u, V_d, V_𝛾},
	}

	return gw, nil
}

func (gw *GraphWalker) EnumPureParticles(Nv_hi int) {

	for i := 1; i <= Nv_hi; i++ {
		gw.targetVtxCount = VtxID(i)
		if i == 1 {
			// Enum base case: Add the only 1x1 (positive) particle
			gw.onParticleCompleted(
				[]EncoderCmd{NewAddVtxCmd(V_e)},
			)
		} else {
			gw.enumAllParticles(gw.encoderScrap[:0], gw.openSlotsScrap[:0], 0, 0)
		}
	}

	gw.EnumStream.Close()
	// 	for ID, pname := range pxs {
	// 		fmt.Printf("   %5d, %4d, %s\n", ID+1, gw.particleCatalog[pname], pname)
	// 	}
	// }
}

// Goal: "complete" the particle while adding the minimum number of vertexes possible.
// Pre: every possible edge "wiring" of the given open edge slots has already been walked or will be walked.
func (gw *GraphWalker) enumAllParticles(
	cmds []EncoderCmd,
	openSlots OpenEdgeSlots,
	numOpenEdgeSlots int32,
	numVtxAdded VtxID,
) {

	// Diagnostic
	if numOpenEdgeSlots != int32(openSlots.CountOpenEdgeSlots()) {
		panic("numOpenEdgeSlots check failed")
	}

	if numVtxAdded == gw.targetVtxCount {
		if numOpenEdgeSlots == 0 {
			gw.onParticleCompleted(cmds)
		} // else {
		// TODO push encoding state and resume on v+1
		//}
		return
	}
	if numOpenEdgeSlots == 0 && numVtxAdded < gw.targetVtxCount && numVtxAdded > 1 {

		// If we're here, we completed a graph but still have more verts to add (which is more than a single particle!)
		return
	}

	// In order to consume as many Vtx as "quickly" as possible, when placing a new vertex, use u, d, and then 𝛾 (in that order)
	numVtxAdded++
	newVtxID := numVtxAdded
	cmds = append(cmds, 0)
	openSlots = append(openSlots, 0)
	for _, vtxType := range gw.vtxChoices {
		cmds[len(cmds)-1] = NewAddVtxCmd(vtxType)
		//numEdgesToWire := vtxType.NumEdges()
		newVtxSlots := NewSlotsForVtxType(newVtxID, vtxType)
		openSlots[len(openSlots)-1] = newVtxSlots
		// numVertsOpen := VtxID(len(openSlots))-1    // minus once since we don't want to count the newly added vertex
		numNowOpen := numOpenEdgeSlots + int32(newVtxSlots.NumOpenEdgeSlots())
		// To maintain the recursive invariant (of every edge writing being tried), permute every possible wiring combo of the newly added vertex.
		// For a single edge ("u") vertex, a single pass through all open vertices covers the span of possible configurations.
		// For a double edge ("d") vertex, a double nested loop is needed to span all possible edge assignments.
		// For a triple edge ("𝛾") vertex, a triple nested loop is needed to span all possible edge assignments.
		// TODO: test if enumeration depends on vtx permutation dir (it shouldn't)

		gw.enumAllEdgeCombos(cmds, openSlots, numNowOpen, numVtxAdded, 1)

	}
}

func (gw *GraphWalker) enumAllEdgeCombos(
	cmds []EncoderCmd,
	openSlots OpenEdgeSlots,
	numOpenEdgeSlots int32,
	numVtxAdded VtxID,
	startVtxID VtxID,
) {

	newVtxID := VtxID(len(openSlots))

	newEdgesToWire := openSlots[newVtxID-1].NumOpenEdgeSlots()

	// Add space for a new "add edge" command
	cmds = append(cmds, 0)

	edgesFound := int32(0)

	// Match the newly added vertex (newVtxID) with all other possible open edges
	for vj := startVtxID; vj < newVtxID; vj++ {
		slot_vj := openSlots[vj-1]
		update, used := slot_vj.UseSlot(1)
		if used {
			// Diagnostic
			if vj != update.VtxID() {
				panic("VtxID check failed")
			}
			cmds[len(cmds)-1] = NewAddEdgeCmd(vj, newVtxID)
			openSlots[vj-1] = update
			openSlots[newVtxID-1] = openSlots[newVtxID-1] - 1
			edgesFound++

			if newEdgesToWire > 1 {
				gw.enumAllEdgeCombos(cmds, openSlots, numOpenEdgeSlots-2, numVtxAdded, vj)
			}

			gw.enumAllParticles(cmds, openSlots, numOpenEdgeSlots-2, numVtxAdded)

			openSlots[vj-1] = slot_vj
			openSlots[newVtxID-1] = openSlots[newVtxID-1] + 1
		}
	}

	// After adding a vertex, if we found no more edges to wire up, we have exhausted all options
	if edgesFound == 0 {
		gw.enumAllParticles(cmds, openSlots, numOpenEdgeSlots, numVtxAdded)
	}
}

func (gw *GraphWalker) onParticleCompleted(cmds []EncoderCmd) {
	X := NewGraph(nil)
	X.AssignFromCmds(cmds)

	// Since the particle enumeration process only makes positive edges, a Traces vector uniquely identifies
	//     a graph and we are thus spared from doing a full canonicalization in order to detect duplicate enumerations.
	//
	// However, for testing, we emit all particles.
	if true { //|| gw.emitted.TryAdd(X.Traces(0)) {
		gw.EnumStream.Outlet <- X
	}
}
