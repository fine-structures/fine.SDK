package lib2x3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// GroupID is a one-based index representing a group (vertex ordinality)  ID.
type GroupID byte

func (g GroupID) GroupRune() byte {
	if g > 0 {
		return 'A' - 1 + byte(g)
	}
	return '?'
}

// Note that all Encodings have an implied "anti-matter" phase, which just flips all the signs.
type TriGraphEncoderOpts int

const (
	IncludeSignModes TriGraphEncoderOpts = 1 << iota
	//TracesAndModesAndCGE
)

func chopBuf(consume []int64, N int32) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}

// GraphTriID is a 2x3 cycle spectrum encoding
type GraphTriID []byte

var (
	ErrNilGraph = errors.New("nil graph")
	ErrBadEdges = errors.New("edge count does not correspond to vertex count")
)

type vtxStatus int32

const (
	newlyReset vtxStatus = iota + 1
	calculating
	canonized
)

type graphState struct {
	vtxCount  int32
	vtxDimSz  int32
	vtxByID   []*graphVtx // vtx by initial vertex ID
	vtx       []*graphVtx // ordered list of vtx groups
	numGroups GroupID     // number of unique vertex groups present

	curCi  int32
	traces Traces
	status vtxStatus
}

// triVtx starts as a vertex in a conventional "2x3" vertex+edge and used to derive a canonical LSM-friendly encoding (i.e. TriID)
// El Shaddai's Grace abounds.  Emmanuel, God with us!  Yashua has come and victory has been sealed!
type triVtx struct {
	GroupID              // which group this is
	VtxType              // which type of vertex this is
	vtxID   byte         // Initial vertex ID (zero-based index)
	edges   [3]groupEdge // Edges to other vertices
}

type groupEdge struct {
	GroupEdge      // Baked sign, edge type, and source group ID (known after canonicalization via cycle spectrum sort)
	srcVtx    byte // initial source vertex index (zero-based)
	isLoop    bool // true if this edge is a loop
	sign      int8 // +1 or -1
}

// Returns:
//    0 if loop
//    1 if edge (non-loop)
func (e *groupEdge) LoopBit() int32 {
	if e.isLoop {
		return 0
	}
	return 1
}

type graphVtx struct {
	triVtx
	cycles []int64 // for traces cycle fingerprint for cycles ci
	Ci0    []int64 // matrix row of X^i for this vtx -- by initial vertex ID
	Ci1    []int64 // matrix row of X^(i+1) for this vtx,
}

type VtxTrait int

const (
	VtxTrait_GroupID VtxTrait = iota
	VtxTrait_EdgesFrom
	VtxTrait_EdgesNormalizedSign
	VtxTrait_EdgesType
	VtxTrait_EdgesSign
)

func (v *triVtx) appendTrait(io []byte, trait VtxTrait, ascii bool) []byte {
	switch trait {
	case VtxTrait_GroupID:
		io = append(io, v.GroupRune())
	case VtxTrait_EdgesFrom:
		for _, e := range v.edges {
			io = append(io, e.GroupRune())
		}
	case VtxTrait_EdgesType:
		if ascii {
			for _, e := range v.edges {
				io = append(io, e.EdgeTypeRune())
			}
		} else {
			ord := byte(0)
			for _, e := range v.edges {
				ord *= NumEdgeTypes
				ord += byte(e.EdgeTypeOrd())
			}
			io = append(io, ord)
		}
	case VtxTrait_EdgesNormalizedSign, VtxTrait_EdgesSign:
		ord := byte(0)
		for _, e := range v.edges {
			sign := byte('+')
			switch trait {
			case VtxTrait_EdgesNormalizedSign:
				if e.IsVtxLoop() && e.IsNeg() {
					sign = '-'
				}
			case VtxTrait_EdgesSign:
				if e.IsNeg() {
					sign = '-'
				}
			}
			if ascii {
				io = append(io, sign)
			} else {
				ord *= 3
				if sign == '+' {
					ord += 1
				} else if sign == '-' {
					ord += 2
				}
			}
		}
		if !ascii {
			io = append(io, ord)
		}
	}
	return io
}

func (v *graphVtx) AddLoop(from int32, edgeSign int8) {
	v.AddEdge(from, edgeSign, true)
}

func (v *graphVtx) AddEdge(from int32, edgeSign int8, isLoop bool) {
	var ei int
	for ei = range v.edges {
		if v.edges[ei].sign == 0 {
			v.edges[ei] = groupEdge{
				srcVtx: byte(from),
				isLoop: isLoop,
				sign:   edgeSign,
			}
			break
		}
	}

	if ei >= 3 {
		panic("tried to add more than 3 edges")
	}
}

func (v *graphVtx) Init(vtxID byte) {
	v.vtxID = vtxID
	v.edges[0].sign = 0
	v.edges[1].sign = 0
	v.edges[2].sign = 0
}

func (X *graphState) reset(numVerts byte) {
	Nv := int32(numVerts)

	X.vtxCount = Nv
	X.numGroups = 0
	X.curCi = 0
	X.status = newlyReset
	if X.vtxDimSz >= Nv {
		return
	}

	// Prevent rapid resize allocs
	if Nv < 8 {
		Nv = 8
	}
	X.vtxDimSz = Nv

	X.vtx = make([]*graphVtx, Nv, 2*Nv)
	X.vtxByID = X.vtx[Nv : 2*Nv]

	// Place cycle bufs on each vtx
	buf := make([]int64, MaxVtxID+3*Nv*Nv)
	X.traces, buf = chopBuf(buf, MaxVtxID)

	for i := int32(0); i < Nv; i++ {
		v := &graphVtx{}
		X.vtxByID[i] = v
		X.vtx[i] = v
		v.Ci0, buf = chopBuf(buf, Nv)
		v.Ci1, buf = chopBuf(buf, Nv)
		v.cycles, buf = chopBuf(buf, Nv)
	}

}

func (X *graphState) AssignGraph(Xsrc *Graph) error {
	if Xsrc == nil {
		X.reset(0)
		return ErrNilGraph
	}

	Nv := Xsrc.NumVerts()
	X.reset(Nv)

	// Init vtx lookup map so we can find the group for a given initial vertex idx
	Xv := X.VtxByID()
	for i := byte(0); i < Nv; i++ {
		Xv[i].Init(i + 1)
		X.vtx[i] = Xv[i]
	}

	// First, add edges that connect to the same vertex (loops)
	for i, vi := range Xv {
		vtype := Xsrc.vtx[i]
		for j := vtype.PosLoops(); j > 0; j-- {
			vi.AddLoop(int32(i), +1)
		}
		for j := vtype.NegLoops(); j > 0; j-- {
			vi.AddLoop(int32(i), -1)
		}
	}

	// Second, add edges connecting two different vertices
	for _, edge := range Xsrc.Edges() {
		ai, bi := edge.VtxIdx()
		pos, neg := edge.EdgeType().NumPosNeg()
		for j := pos; j > 0; j-- {
			Xv[ai].AddEdge(bi, +1, false)
			Xv[bi].AddEdge(ai, +1, false)
		}
		for j := neg; j > 0; j-- {
			Xv[ai].AddEdge(bi, -1, false)
			Xv[bi].AddEdge(ai, -1, false)
		}
	}

	// Calculate and assign siblings for every edge
	// This ensures we can sort (group) edges first by co-connectedness
	for _, v := range Xv {
		negLoops := byte(0)
		numEdges := byte(0)
		Ne := byte(0)
		for _, e := range v.edges {
			if e.sign != 0 {
				Ne++
				if e.isLoop {
					if e.sign < 0 {
						negLoops++
					}
				} else {
					numEdges++
				}
			}
		}
		v.VtxType = GetVtxType(negLoops, numEdges)
		if Ne != 3 || v.VtxType == V_nil {
			return ErrBadEdges
		}
	}

	return nil
}

func (X *graphState) Vtx() []*graphVtx {
	return X.vtx[:X.vtxCount]
}

func (X *graphState) VtxByID() []*graphVtx {
	return X.vtxByID[:X.vtxCount]
}

func (X *graphState) sortVtxGroups() {
	Xg := X.Vtx()

	// With edges on vertices now canonic order, we now re-order to assert canonic order within each group.
	sort.Slice(Xg, func(i, j int) bool {
		vi := Xg[i]
		vj := Xg[j]

		// if d := vi.VtxType - vj.VtxType; d != 0 {
		// 	return d < 0
		// }

		// Keep groups together
		if d := int(vi.GroupID) - int(vj.GroupID); d != 0 {
			return d < 0
		}

		for ei := range vi.edges {
			if d := vi.edges[ei].Ord() - vj.edges[ei].Ord(); d != 0 {
				return d < 0
			}
		}

		return false
	})
}

// For the currently assigned Graph, this calculates its cycles and traces up to a given level.
func (X *graphState) calcCyclesUpTo(numTraces int32) {
	Nv := X.vtxCount

	if numTraces < Nv {
		numTraces = Nv
	}

	Xv := X.VtxByID()

	// Init C0
	if X.status == newlyReset {
		X.status = calculating

		for i, vi := range Xv {
			for j := 0; int32(j) < Nv; j++ {
				c0 := int64(0)
				if i == j {
					c0 = 1
				}
				vi.Ci0[j] = c0
			}
		}
	}

	// This loop effectively calculates each successive graph matrix power.
	for ; X.curCi < numTraces; X.curCi++ {
		ci := X.curCi
		odd := (ci & 1) != 0
		traces_ci := int64(0)

		// Calculate Ci+1 by "flowing" the current state (Ci) through X's edges.
		for _, vi := range Xv {

			// Alternate which is the prev / next state store
			Ci0, Ci1 := vi.Ci0, vi.Ci1
			if odd {
				Ci0, Ci1 = Ci1, Ci0
			}

			for j := int32(0); j < Nv; j++ {
				dot := int64(0)
				vj := Xv[j]
				for _, e := range vj.edges {
					input := Ci0[e.srcVtx]
					if e.sign < 0 {
						input = -input
					}
					dot += input
				}
				Ci1[j] = dot
			}

			vi_cycles_ci := Ci1[vi.vtxID-1]
			if ci < Nv {
				vi.cycles[ci] = vi_cycles_ci
			}
			traces_ci += vi_cycles_ci
		}
		X.traces[ci] = traces_ci
	}

}

func (X *graphState) canonize() {
	if X.status >= canonized {
		return
	}

	{
		Nv := X.vtxCount
		X.calcCyclesUpTo(Nv)

		Xv := X.Vtx()

		// Two major sub steps:
		//    I)  Edge normalization: turn pos+neg loop pairs into group edges wherever possible
		//    II) Vertex-pair normalization: for every edge, normalize to factor complimentary group IDs from
		// Look for adjacent vtx that can be consolidated into an equivalent single cycle group
		//  e.g.    1^-~2     => 1^-2^,
		//          1^-~2-3=4 => 1^-2-3-4-2,1-4
		// X=1-2=Y   => X-1-Y, X-2-Y,
		{
			// (1) Look for edge pairs on two adjacent vertices
			// The conversion we're looking for is two vertices that, when combined,
			{

			}

			// (2) If remaining edge points to the other vtx, then these two vtx are the same cycle group since B+XX + A+YY <=> 2(A'XY)
			// This means the shared edge becomes a double edge in the new cycle group of these 2 vtx.
			{

			}
		}

		// Sort vertices by vertex's innate characteristics & cycle signature
		sort.Slice(Xv, func(i, j int) bool {
			vi := Xv[i]
			vj := Xv[j]

			// Sort by cycle count first and foremost
			// The cycle count vector (an integer sequence of size Nv) is what characterizes a vertex.
			for ci := int32(0); ci < Nv; ci++ {
				d := vi.cycles[ci] - vj.cycles[ci]
				if d != 0 {
					return d < 0
				}
			}
			return false
		})

		// Now that vertices are sorted by cycle vector, assign each vertex the cycle group number now associated with its vertex index.
		X.numGroups = 1
		var v_prev *graphVtx
		for _, v := range Xv {
			if v_prev != nil {
				for ci := int32(0); ci < Nv; ci++ {
					if v.cycles[ci] != v_prev.cycles[ci] {
						X.numGroups++
						break
					}
				}
			}
			v.GroupID = X.numGroups
			v_prev = v
		}

		// With cycle group numbers assigned to each vertex, assign srcGroup to all edges and then finally order edges on each vertex canonically.
		for _, v := range Xv {
			for ei, e := range v.edges {
				src_vi := X.vtxByID[e.srcVtx]
				v.edges[ei].GroupEdge = FormGroupEdge(src_vi.GroupID, src_vi.GroupID == v.GroupID, e.isLoop, e.sign < 0)
			}

			// With each edge srcGroup now assigned, we can order the edges canonically
			//
			// Canonically order edges by edge type then by edge sign & groupID
			// Note that we ignore edge grouping, which means graphs with double edges will encode onto graphs having a non-double edge equivalent.
			// For v=6, this turns out to be 4% less graph encodings (52384 vs 50664) and for v=8 about 2% less encodings (477k vs 467k).
			// If we figure out we need these encodings back (so they count as valid state modes), we can just export the grouping info.
			//
			// A fundamental mystery is: what ARE these particle "sign modes" anyway??
			// Are they perhaps vibration modes of a membrane (and so all are valid)?
			// Or are Griggs-style graphs merely a stepping stone to understanding the underlying symbolic representation of God's particles that
			//     are not bound to the ways a human-conceived graph framework can construct graphs that reduces to a particular particle representation.
			sort.Slice(v.edges[:], func(i, j int) bool {
				d := v.edges[i].Ord() - v.edges[j].Ord()
				return d < 0
			})
		}

		X.sortVtxGroups()
	}

	X.status = canonized
}

/*
func (X *graphState) findGroupRuns(io []uint8) []uint8 {
	Xg := X.Vtx()
	N := uint8(len(Xg))
	runLen := uint8(0)

	for gi := uint8(0); gi < N; gi += runLen {
		viGroup := Xg[gi].GroupID
		runLen = 1
		for gj := gi + 1; gj < N; gj++ {
			vjGroup := Xg[gj].GroupID
			if viGroup != vjGroup {
				break
			}
			runLen++
		}
		io = append(io, runLen)
	}
	return io
}
*/

/*
	    Number of families
	    |   Family edges [*^ABCDEF..]
	    |   |   Family edges counts
	    |   |   |               Cycles composition
	    |   |   |               |           Edges sign modulation modes (8 counts)
		|   |   |               |           |
  UTF8:	2   *AC 2  *^B 2 ...    BB. AC.     N--- N--+ N-+- N-++ N+-- N+-+ N++- N+++
  Bits: 6   666 2  666 6        66  66      6    6    6    6    6    6    6    6



*/

func (X *graphState) getTraitRun(Xv []*graphVtx, vi int, trait VtxTrait) int { //(triID []byte, modes []byte) {
	Nv := len(Xv)
	var buf [8]byte

	viTr := Xv[vi].appendTrait(buf[:0], trait, false)

	runLen := 1
	for vj := vi + 1; vj < Nv; vj++ {
		vjTr := Xv[vj].appendTrait(buf[4:4], trait, false)
		if !bytes.Equal(viTr, vjTr) {
			break
		}
		runLen++
	}

	return runLen
}

type GraphEncodingOpts int32

const (
	EncodeHumanReadable GraphEncodingOpts = 1 << iota
	EncodeProperties
	EncodeState
	___                = 1
	GraphEncodingMaxSz = (1 + ___ + 4*MaxVtxID + ___ + MaxVtxID + ___ + MaxVtxID*(3+___) + ___ + 8*(3+___) + 0xF) &^ 0xF
)

func (X *graphState) AppendGraphEncoding(io []byte, opts GraphEncodingOpts) []byte {
	X.canonize()
	return X.appendGraphEncoding(io, opts)
}

type tracesGroupComponent struct {
	count [NumEdgeTypes]int8
}

func (tg *tracesGroupComponent) tracesCoeffs() (loopNet, groupNet, edgeAbs int8) {
	loopNet = tg.count[LocalLoop_Pos] - tg.count[LocalLoop_Neg]
	
	groupNet = tg.count[GroupEdge_Pos] - tg.count[GroupEdge_Neg]
	if groupNet < 0 {
		groupNet = -groupNet
	}
	
	edgeAbs = tg.count[BasicEdge_Pos] - tg.count[BasicEdge_Neg]
	if edgeAbs < 0 {
		edgeAbs = -edgeAbs
	}
	
	return
}

func (tg *tracesGroupComponent) Init() (sum int8) {
	for i := range tg.count {
		tg.count[i] = 0
	}
	return
}


func (X *graphState) appendGraphEncoding(io []byte, opts GraphEncodingOpts) []byte { //(triID []byte, modes []byte) {
	// Generally, there's plenty of room (even accounting for adding spaces in ascii mode)
	// enc := io[len(io):cap(io)]
	// if len(enc) < GraphEncodingMaxSz {
	// 	enc = make([]byte, len(io) + GraphEncodingMaxSz)
	// 	copy(enc, io)
	// }

	//var buf [32]byte

	Xv := X.Vtx()
	//Nv := len(Xv)
	ascii := (opts & EncodeHumanReadable) != 0

	//traits := make([]VtxTrait, 0, 4)
	
	// Tally all components (all edge classes and signs; 3 types, 2 signs => 6 total)
	Tg := make([]tracesGroupComponent, X.numGroups)
	for _, v := range Xv {
		for _, e := range v.edges { 
			compType := e.EdgeTypeOrd()
			if e.GroupID() == 0 {
				panic("unassigned edge group")
			}
			Tg[e.GroupID()-1].count[compType]++
		}
	}
		
	if (opts & EncodeProperties) != 0 {
		for gi, g := range Tg {
			loopNet, groupNet, edgeAbs := g.tracesCoeffs()
			if ascii {
				if gi > 0 { io = append(io, ' ') }
				io = append(io, 'A' + byte(gi), '(')

				if loopNet != 0 {
					//if loopNet > 0 { io = append(io, ' ') }
					io = strconv.AppendInt(io, int64(loopNet), 10)
				}
				io = append(io, ',')
				if groupNet != 0 {
					//if groupNet > 0 { io = append(io, ' ') }
					io = strconv.AppendInt(io, int64(groupNet), 10)
				}
				io = append(io, ',')
				if edgeAbs != 0 {
					//if edgeAbs > 0 { io = append(io, ' ') }
					io = strconv.AppendInt(io, int64(edgeAbs), 10)
				}
				io = append(io, ')')

			} else {
				io = append(io,
					MaxVtxID + byte(loopNet),
					MaxVtxID + byte(groupNet),
					MaxVtxID + byte(edgeAbs))
			}
		}
		// traits = append(traits,
		// 	VtxTrait_EdgesFrom,
		// 	VtxTrait_EdgesSign,
		// )
	}

	if (opts & EncodeState) != 0 {
		// traits = append(traits,
		// 	VtxTrait_EdgesType,
		// 	VtxTrait_EdgesSign,
		// )
		
		for gi, g := range Tg {
			if ascii {
				if gi > 0 { io = append(io, ' ') }
				io = append(io, 'A' + byte(gi), '(')
				
				if g.count[LocalLoop_Pos] + g.count[LocalLoop_Neg] > 0 {
					if c := g.count[LocalLoop_Pos]; c >= 0 {
						io = strconv.AppendInt(io, int64(c), 10)
					} else {
						io = append(io, ' ')
					}
					if c := g.count[LocalLoop_Neg]; c > 0 {
						io = strconv.AppendInt(io, int64(-c), 10)
					} else {
						io = append(io, '-', '0')
					}
				}
				
				io = append(io, ',')

				if g.count[GroupEdge_Pos] + g.count[GroupEdge_Neg] > 0 {
					if c := g.count[GroupEdge_Pos]; c >= 0 {
						io = strconv.AppendInt(io, int64(c), 10)
					} else {
						io = append(io, ' ')
					}
					if c := g.count[GroupEdge_Neg]; c > 0 {
						io = strconv.AppendInt(io, int64(-c), 10)
					} else {
						io = append(io, '-', '0')
					}
				}
				
				io = append(io, ',')

				if g.count[BasicEdge_Pos] + g.count[BasicEdge_Neg] > 0 {
					if c := g.count[BasicEdge_Pos]; c >= 0 {
						io = strconv.AppendInt(io, int64(c), 10)
					} else {
						io = append(io, ' ')
					}
					if c := g.count[BasicEdge_Neg]; c > 0 {
						io = strconv.AppendInt(io, int64(-c), 10)
					} else {
						io = append(io, '-', '0')
					}
				}
				
				io = append(io, ')')
			} else {
				io = append(io,
					byte(g.count[LocalLoop_Neg]), byte(g.count[LocalLoop_Pos]),
					byte(g.count[GroupEdge_Neg]), byte(g.count[GroupEdge_Pos]),
					byte(g.count[BasicEdge_Neg]), byte(g.count[BasicEdge_Pos]),
				)
					
				for _, c := range g.count {
					io = append(io, byte(c))
				}
			}
		}
	}

	// for _, ti := range traits {
	// 	runLen := 0
	// 	RLE := buf[:0]
	// 	for vi := 0; vi < Nv; vi += runLen {
	// 		runLen = X.getTraitRun(Xv, vi, ti)

	// 		// For readability, print the family count first in ascii mode (but for LSM it follows the edges)
	// 		if ascii {
	// 			if runLen == 1 {
	// 				io = append(io, ' ')
	// 			} else {
	// 				io = strconv.AppendInt(io, int64(runLen), 10)
	// 			}
	// 		}

	// 		io = Xv[vi].appendTrait(io, ti, ascii)
	// 		if ascii {
	// 			io = append(io, ' ')
	// 		} else {
	// 			RLE = append(RLE, byte(runLen))
	// 		}
	// 	}

	// 	io = append(io, RLE...)
	// }

	return io
}

type edgeTrait int

const (
	kVtxID = iota
	kEdgeHomeGroup
	kEdgeType
	kEdgeFrom
	kEdgeSign

	kNumLines = 5
)

func (v *triVtx) printEdgesDesc(vi int, trait edgeTrait, dst []byte) {

	switch trait {
	case kVtxID:
		vid := vi + 1
		if vid > 9 {
			dst[0] = '0' + byte(vid/10)
			dst[1] = '0' + byte(vid%10)
		} else {
			dst[0] = ' '
			dst[1] = '0' + byte(vid)
		}
		dst[2] = ' '
	default:
		for ei, e := range v.edges {
			r := byte('!')
			switch trait {
			case kEdgeFrom:
				r = e.FromGroupRune()
			case kEdgeType:
				r = e.EdgeTypeRune()
			case kEdgeSign:
				r = e.SignRune(true)
			case kEdgeHomeGroup:
				r = v.GroupRune()
			}
			dst[ei] = r
		}
	}
}

// Note that this graph is not assumed to be in a canonic state -- this proc ust visualizes what's there.
func (X *graphState) PrintVtxGrouping(out io.Writer) {
	X.canonize()

	labels := []string{
		"      V       ",
		"    GROUP     ",
		"  EDGE TYPE   ",
		"  EDGE FROM   ",
		"  EDGE SIGN   ",
	}

	Nv := int(X.vtxCount)

	const vtxRad = 2
	vtxWid := 1 + (2 + (2*vtxRad + 1) + 2)
	totWid := (int(X.vtxCount) * vtxWid) + 1

	marginL := len(labels[0])
	bytesPerRow := marginL + totWid + 1
	rows := make([]byte, kNumLines*bytesPerRow)
	for i := range rows {
		rows[i] = ' '
	}

	for li := 0; li < kNumLines; li++ {
		row := rows[li*bytesPerRow:]
		copy(row, labels[li])
		row = row[marginL:]

		switch li {
		// case kEdgeFrom:
		// 	xi := vtxWid / 2
		// 	for vi := 0; vi < Nv; vi++{
		// 		row[xi] = '.'
		// 		xi += vtxWid
		// 	}
		case kEdgeHomeGroup:
			for xi := 0; xi < totWid; xi++ {
				row[xi] = ':'
			}
		}

		row[totWid] = '\n'
	}

	Xv := X.Vtx()

	var viEdges, vjEdges [3]byte

	for li := 0; li < kNumLines; li++ {
		row := rows[li*bytesPerRow+marginL:]
		runL := 0

		for vi := 0; vi < Nv; {
			vtxRunLen := 1
			traitConst := true

			Xv[vi].printEdgesDesc(vi, edgeTrait(li), viEdges[:])

			for vj := vi + 1; vj < Nv; vj++ {
				if Xv[vi].GroupID != Xv[vj].GroupID {
					break
				}
				Xv[vj].printEdgesDesc(vj, edgeTrait(li), vjEdges[:])
				if viEdges != vjEdges {
					traitConst = false
				}
				vtxRunLen++
			}

			runR := vtxWid * (vi + vtxRunLen)
			runC := (runL + runR) >> 1

			// Draw vertical lines along the left and right of the run
			{
				if vi == 0 {
					row[0] = ':'
				}
				row[runR] = ':'
			}

			traitRun := traitConst && (li == kEdgeFrom || li == kEdgeType || li == kEdgeSign)

			for vj := vi; vj < vi+vtxRunLen; vj++ {
				vtxC := vtxWid*vj + vtxWid>>1

				if !traitRun {
					switch li {
					case kEdgeHomeGroup:
					default:
						Xv[vj].printEdgesDesc(vj, edgeTrait(li), vjEdges[:])
						copy(row[vtxC-1:], vjEdges[:])
					}
				}
			}

			switch li {
			case kEdgeHomeGroup:
				c := Xv[vi].GroupRune()
				for w := runL + 3; w <= runR-3; w++ {
					row[w] = c
				}
			}

			if traitRun {
				copy(row[runC-1:], viEdges[:])
			}

			runL = runR
			vi += vtxRunLen
		}

	}

	out.Write(rows)
}

func (X *graphState) PrintCycleSpectrum(out io.Writer) {
	X.canonize()

	Nv := X.vtxCount
	Xv := X.Vtx()

	for ci := int32(0); ci < Nv; ci++ {
		fmt.Fprintf(out, "%8d C%-2d", X.traces[ci], ci+1)
		for _, vi := range Xv {
			fmt.Fprintf(out, "%8d  ", vi.cycles[ci])
		}
		out.Write([]byte{'\n'})
	}
}

func (X *graphState) Traces(numTraces int) Traces {
	if numTraces <= 0 {
		numTraces = int(X.vtxCount)
	}

	X.calcCyclesUpTo(int32(numTraces))
	return X.traces[:numTraces]
}
