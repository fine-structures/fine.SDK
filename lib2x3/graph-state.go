package lib2x3

import (
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// EdgeCode is a code spans 1.. that encodes the edge sign, source group ID (if non-loop) or loop sign (if loop)
type EdgeCode byte

const (
	//NilEdge    EdgeCode = 0
	PosLoop EdgeCode = 1
	NegLoop EdgeCode = 2
)

type graphEdge struct {
	srcVtx   byte     // index of source vertex
	isLoop   bool     // true if this edge is a loop
	edgeSign int8     // +1 or -1
	edgeCode EdgeCode // Canonic edge info specifying edge sign and src groupID (group 0 denotes loop)
}

func (ord EdgeCode) IsNegEdge() bool {
	return (ord & 1) == 0
}

func (ord EdgeCode) SignAndGroup() (edgeSign int8, srcGroup byte) {
	edgeSign = int8(1)
	if (ord & 1) == 0 {
		edgeSign = -1
	}
	srcGroup = byte(ord-1) >> 1
	return
}

func (ord EdgeCode) WriteStr(io []byte) []byte {
	var str [4]byte
	i := 2

	// May the Lord have mercy on us for we know not what we do.
	sign, srcGroup := ord.SignAndGroup()
	if sign < 0 {
		str[0] = '-'
	} else {
		str[0] = '+'
	}
	if srcGroup == 0 {
		str[1] = 'o' // fancy symbol for fancy zero that straddles true zero.  Emmanuel!
	} else {
		if srcGroup >= 10 {
			str[1] = '0' + (srcGroup / 10)
			i++
		}
		str[i-1] = '0' + (srcGroup % 10)
	}

	return append(io, str[:i]...)
}

// Pre: srcGroup > 0
func (e *graphEdge) EncodeSrcGroup(srcGroup byte) {

	if e.isLoop {
		srcGroup = 0
	}

	// Make positive edges to appear lexicographically before negative edges, so make positive negative
	// in:    +0   -0   +1   -1   +2   -2
	// ord:    1    2    3    4    5    6
	ord := (srcGroup << 1) + 1
	if e.edgeSign < 0 {
		ord += 1
	}

	e.edgeCode = EdgeCode(ord)
}

func (e graphEdge) WriteStr(io []byte) []byte {
	return e.edgeCode.WriteStr(io)
}

type graphVtx struct {
	groupID byte // cycle group one-based index (known after cycle spectrum is computed)
	runLen  byte // vertex run len for this groupID
	edges   [3]graphEdge
	traces  []traceStep // for traces cycle fingerprint for cycles ci
}

func (v *graphVtx) VtxType() VtxType {
	numNegLoops := byte(0)
	numEdges := byte(3)
	for _, e := range v.edges {
		switch e.edgeCode {
		case PosLoop:
			numEdges--
		case NegLoop:
			numNegLoops++
			numEdges--
		}
	}
	return GetVtxType(numNegLoops, numEdges)
}

func (v *graphVtx) EncodeTo(in []byte) []byte {
	return append(in,
		0xFF-v.runLen, // make largest group run lens rank first
		byte(v.edges[0].edgeCode),
		byte(v.edges[1].edgeCode),
		byte(v.edges[2].edgeCode))

}

func (v *graphVtx) AddLoop(from int32, edgeSign int8) {
	v.AddEdge(from, edgeSign, true)
}

func (v *graphVtx) AddEdge(from int32, edgeSign int8, isLoop bool) {

	var ei int
	for ei = range v.edges {
		if v.edges[ei].edgeSign == 0 {
			v.edges[ei] = graphEdge{
				srcVtx:   byte(from),
				isLoop:   isLoop,
				edgeSign: edgeSign,
			}
			break
		}
	}

	if ei >= 3 {
		panic("tried to add more than 3 edges")
	}

}

func (v *graphVtx) Clear() {
	v.runLen = 1
	v.edges[0].edgeSign = 0
	v.edges[1].edgeSign = 0
	v.edges[2].edgeSign = 0
}

type traceStep struct {
	cycles int64    // vtx cycle count
	extSum int64    // vtx cycle sum from non-adjacent vertices
	input  [3]int64 // vtx cycle count incoming from edges
}

const Graph3IDSz = 16

type Graph3ID [Graph3IDSz]byte

func (uid Graph3ID) String() string {
	return Base32Encoding.EncodeToString(uid[:])
}

// GeohashBase32Alphabet is the alphabet used for Base32Encoding
const GeohashBase32Alphabet = "0123456789bcdefghjkmnpqrstuvwxyz"

var (
	// Base32Encoding is used to encode/decode binary buffer to/from base 32
	Base32Encoding = base32.NewEncoding(GeohashBase32Alphabet).WithPadding(base32.NoPadding)
)

func chopBuf(consume []int64, N int32) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}

type vtxStatus int32

const (
	newlyReset vtxStatus = iota + 1
	calculating
	canonized
)

type graphState struct {
	vtxCount       int32
	vtxDimSz       int32
	vtx            []*graphVtx
	numCycleGroups byte
	numVtxGroups   byte

	curCi  int32
	Ci0    []int64
	Ci1    []int64
	traces Traces
	triID  []byte
	status vtxStatus
}

func (X *graphState) reset(numVerts byte) {
	Nv := int32(numVerts)

	X.vtxCount = Nv
	X.numVtxGroups = numVerts
	X.curCi = 0
	X.numCycleGroups = 0
	X.triID = X.triID[:0]
	X.status = newlyReset
	if X.vtxDimSz >= Nv {
		return
	}

	// Prevent rapid resize allocs
	if Nv < 8 {
		Nv = 8
	}
	X.vtxDimSz = Nv

	const numTraces = 16
	VxV := Nv * Nv
	X.vtx = make([]*graphVtx, Nv)
	X.triID = make([]byte, 0, 20+4*Nv)
	for i := range X.vtx {
		X.vtx[i] = &graphVtx{}
	}

	// Place cycle buf on each vtx
	cycleInfo := make([]traceStep, VxV) //+numTraces)
	for i := int32(0); i < Nv; i++ {
		X.vtx[i] = &graphVtx{
			traces: cycleInfo[i*Nv : (i+1)*Nv],
		}
	}

	buf := make([]int64, 2*VxV+numTraces)

	// The number of total int64 needed are two Nv x Nv (triangular) matrices (Ci0 amd Ci1)
	// TODO: no need to alloc lower triangle
	X.Ci0, buf = chopBuf(buf, VxV)
	X.Ci1, buf = chopBuf(buf, VxV)

	X.traces = buf
}

func (X *graphState) Validate() error {
	Ne := int32(0)
	Nv := X.vtxCount
	for _, vi := range X.VtxGroups() {
		for _, e := range vi.edges {
			if e.edgeSign != 0 {
				Ne += int32(vi.runLen)
			}
		}
	}
	if Ne != 3*Nv {
		return errors.New("incomplete graph")
	}

	return nil
}

func (X *graphState) AssignGraph(Xsrc *Graph) {
	if Xsrc == nil {
		X.reset(0)
		return
	}

	Nv := Xsrc.NumVerts()
	X.reset(Nv)

	// First, add sedges that connect to the same vertex (loops and arrows)
	for i, vi := range Xsrc.Vtx() {
		vtx := X.vtx[i]
		vtx.Clear()
		for j := vi.NumLoops(); j > 0; j-- {
			vtx.AddLoop(int32(i), +1)
		}
		for j := vi.NumArrows(); j > 0; j-- {
			vtx.AddLoop(int32(i), -1)
		}
	}

	// Second, add edges connecting two different vertices
	for _, edge := range Xsrc.Edges() {
		ai, bi := edge.VtxIdx()
		pos, neg := edge.EdgeType().NumPosNeg()
		for j := pos; j > 0; j-- {
			X.vtx[ai].AddEdge(bi, +1, false)
			X.vtx[bi].AddEdge(ai, +1, false)
		}
		for j := neg; j > 0; j-- {
			X.vtx[ai].AddEdge(bi, -1, false)
			X.vtx[bi].AddEdge(ai, -1, false)
		}
	}

	if err := X.Validate(); err != nil {
		panic(err)
	}
}

func (X *graphState) VtxGroups() []*graphVtx {
	return X.vtx[:X.numVtxGroups]
}

func (X *graphState) resortVtxGroups() {
	Xvtx := X.VtxGroups()

	// With edges on vertices now canonic order, we now re-order to assert canonic order within each group.
	sort.Slice(Xvtx, func(i, j int) bool {
		vi := Xvtx[i]
		vj := Xvtx[j]

		d := int32(vi.groupID) - int32(vj.groupID)
		if d != 0 {
			return d < 0
		}

		d = int32(vi.runLen) - int32(vj.runLen)
		if d != 0 {
			return d > 0
		}

		// If we're here, we're ordering within a given groupID.
		for ei := range vi.edges {
			d := int(vi.edges[ei].edgeCode) - int(vj.edges[ei].edgeCode)
			if d != 0 {
				return d < 0
			}
		}

		return false
	})
}

// For the currently assigned Graph, this calculates its cycles and traces up to a given level.
func (X *graphState) calcUpTo(numTraces int32) {
	Nv := X.vtxCount

	if numTraces < Nv {
		numTraces = Nv
	}

	Ci0 := X.Ci0
	Ci1 := X.Ci1

	Xvtx := X.vtx[:Nv]

	// Init C0
	if X.status == newlyReset {
		X.status = calculating
		for vi := int32(0); vi < Nv; vi++ {
			for vj := int32(0); vj < Nv; vj++ {
				c0 := int64(0)
				if vj == vi {
					c0 = 1
				}
				Ci0[vi*Nv+vj] = c0
			}
		}
	} else if (X.curCi & 1) != 0 { // Resume from the proper state
		Ci0, Ci1 = Ci1, Ci0
	}

	// This loop effectively calculates each successive graph matrix power.
	for ; X.curCi < numTraces; X.curCi++ {

		// Calculate Ci+1 by "flowing" the current state (Ci) through X's edges.
		// We only need to calculate upper triangle since a graph adjacency matrix is always triangular-symmetric.
		for vi := int32(0); vi < Nv; vi++ {
			for vj := int32(0); vj < Nv; vj++ { // vj := vi; vj < Nv; vj++ {
				dot := int64(0)
				for _, e := range Xvtx[vj].edges {
					src_v := int32(e.srcVtx)
					input := Ci0[vi*Nv+src_v]
					if e.edgeSign < 0 {
						input = -input
					}
					dot += input
				}
				Ci1[vi*Nv+vj] = dot
			}
		}

		// With Ci+1 now calculated, store the Traces for the current cycle level (ci)
		ci := X.curCi
		traces_ci := int64(0)
		calcCycles := ci < Nv

		for vi := int32(0); vi < Nv; vi++ {
			vi_vtx := Xvtx[vi]

			// Total completed cycles of length ci is the sum of the diagonal.
			vi_cycles := Ci1[vi*Nv+vi]
			traces_ci += vi_cycles

			if calcCycles {
				e0 := Ci1[vi*Nv+int32(vi_vtx.edges[0].srcVtx)]
				e1 := Ci1[vi*Nv+int32(vi_vtx.edges[1].srcVtx)]
				e2 := Ci1[vi*Nv+int32(vi_vtx.edges[2].srcVtx)]

				// Sort 3 items (ascending)
				if e0 > e1 {
					e0, e1 = e1, e0
				}
				if e0 > e2 {
					e0, e2 = e2, e0
				}
				if e1 > e2 {
					e1, e2 = e2, e1
				}
				if !(e0 <= e1 && e1 <= e2) {
					panic("bad sort")
				}

				Tci := &vi_vtx.traces[ci]
				Tci.cycles = vi_cycles
				Tci.extSum = 0
				Tci.input[0] = e0
				Tci.input[1] = e1
				Tci.input[2] = e2

				for vj := int32(0); vj < Nv; vj++ {
					switch {
					case vi == vj:
					case vj == int32(vi_vtx.edges[0].srcVtx):
					case vj == int32(vi_vtx.edges[1].srcVtx):
					case vj == int32(vi_vtx.edges[2].srcVtx):
					default:
						Tci.extSum += Ci1[vi*Nv+vj]
					}
				}
			}
		}
		X.traces[ci] = traces_ci

		// On iteration, the "next" cycle state becomes the current state
		Ci0, Ci1 = Ci1, Ci0
	}

}

func (X *graphState) canonize() {
	if X.status >= canonized {
		return
	}

	Nv := X.vtxCount
	X.calcUpTo(Nv)
	vtxOrder := make([]byte, Nv)
	for i := range vtxOrder {
		vtxOrder[i] = byte(i)
	}

	Xvtx := X.VtxGroups()

	// Sort vertices by vertex's innate characteristics & cycle signature
	{
		sort.Slice(vtxOrder, func(i, j int) bool {
			vi := Xvtx[vtxOrder[i]]
			vj := Xvtx[vtxOrder[j]]

			// Sort by cycle count first and foremost
			// The cycle count vector (an integer sequence of size Nv) is what characterizes a vertex.
			for ci := int32(0); ci < Nv; ci++ {
				d := vi.traces[ci].cycles - vj.traces[ci].cycles
				if d != 0 {
					return d < 0
				}
			}

			return false
		})
	}

	// Now that vertices are sorted by cycle vector,  assign each vertex the groupID now associated with its vertex index.
	// With vertices in cycle-spectrum-canonic order we can now assign a groupID to each vertex.
	// The groupID starts with 1 and group 0 is reserved for denote a loop (either positive or negative based on edgeSign).
	{
		X.numCycleGroups = 1
		var v_prev *graphVtx
		for _, vi_idx := range vtxOrder[:Nv] {
			vi := Xvtx[vi_idx]
			if v_prev != nil {
				for ci := int32(0); ci < Nv; ci++ {
					if vi.traces[ci].cycles != v_prev.traces[ci].cycles {
						X.numCycleGroups++
						break
					}
				}
			}
			vi.groupID = X.numCycleGroups
			v_prev = vi
		}
	}

	// With groupIDs assigned to each vertex, assign srcGroup to all edges and order them edges on each vertex canonically
	// Sort via vtxOrder so that .srcVtx still has the correct vtx index
	for _, vi := range Xvtx {
		for ei := range vi.edges {
			srcGroup := Xvtx[vi.edges[ei].srcVtx].groupID
			vi.edges[ei].EncodeSrcGroup(srcGroup)
		}

		// Canonically order edges by edge type then by edge sign & groupID
		sort.Slice(vi.edges[:], func(i, j int) bool {
			return vi.edges[i].edgeCode < vi.edges[j].edgeCode
		})
	}

	X.resortVtxGroups()

	// Last but not least, we can do an RLE-style compression of the now-canonic vertex series.
	// Note that doing so invalidates edge.srvVtx values, so lets zero them out for safety.
	// Work right to left
	{
		L := byte(0)
		for R := int32(1); R < Nv; R++ {
			Lvtx := Xvtx[L]
			Rvtx := Xvtx[R]
			identical := false
			if Lvtx.groupID == Rvtx.groupID {
				identical = true
				for ei := range Lvtx.edges {
					if Lvtx.edges[ei].edgeCode != Rvtx.edges[ei].edgeCode {
						identical = false
						break
					}
				}
			}

			// If exact match, absorb R into L, otherwise advance L (old R becomes new L)
			if identical {
				Lvtx.runLen += Rvtx.runLen
			} else {
				L++
				Xvtx[L], Xvtx[R] = Xvtx[R], Xvtx[L]
			}
		}
		X.numVtxGroups = L + 1
	}

	{
		Nv := byte(X.vtxCount)
		triID := append(X.triID[:0], Nv)
		for _, gi := range X.VtxGroups() {
			Nv -= gi.runLen
			triID = gi.EncodeTo(triID)
		}
		if Nv != 0 {
			panic("final compaction chk failed")
		}
		X.triID = triID
	}

	// Now that we have consolidated identical vertices, do final resort to move vtx groups with highest runLen first
	X.resortVtxGroups()

	X.status = canonized
}

// GraphTriID is a 2x3 cycle spectrum encoding
type GraphTriID []byte

func (triID GraphTriID) String() string {
	out := &strings.Builder{}
	fmt.Fprintf(out, "v=%d ", triID[0])
	for i := 1; i < len(triID); i += 4 {
		fmt.Fprintf(out, "%d[%d:%d:%d] ", triID[i], triID[i+1], triID[i+2], triID[i+3])
	}
	return out.String()
}

func (X *graphState) GraphTriID() GraphTriID {
	X.canonize()
	return X.triID
}

// func (X *graphState) AppendCycleSpectrumTriID(io []byte) ([]byte) {
// 	X.canonize()
// 	io = append(io, byte(X.vtxCount))
// 	io = append(io, X.triID...)
// 	return io
// }

/*
func (X *graphState) computeUID() {
	if X.status >= uidComputed {
		return
	}

	X.canonize()

	{
		hasher := splitMix{}
		hasher.Reset()

		{
			Nv := X.vtxCount
			for _, vi := range X.vtx[:Nv] {
				hasher.Write(int64(vi.groupID))
				for _, e := range vi.edges {
					hasher.Write(int64(e.edgeCode))
				}
				// for ci := int32(0); ci < Nv; ci++ {
				// 	// Open mystery: removing [0:3]input (below) decreases total graph count as expected, but it also decreases the unique Traces count!?!
				// 	// The unique Traces count of a complete catalog should be *unaffected* by GraphUID choices and conventions, no!?
				// 	hasher.Write(vi.traces[ci].cycles)
				// 	hasher.Write(vi.traces[ci].input[0])
				// 	hasher.Write(vi.traces[ci].input[1])
				// 	hasher.Write(vi.traces[ci].input[2])
				// 	hasher.Write(vi.traces[ci].extSum)
				// 	hasher.Write(vi.VtxType())
				// }
			}
		}

		hash := hasher.Sum()
		copy(X.uid[:], hash[:])
		X.status = uidComputed
	}
}
*/

func (X *graphState) PrintCycleSpectrum(out io.Writer) {
	X.canonize()

	Nv := X.vtxCount

	Xvtx := X.VtxGroups()
	var buf [16]byte

	// Cycle Group info
	fmt.Fprintf(out, "\n                  ")
	for _, vi := range Xvtx {
		fmt.Fprintf(out, " %2dx @%-2d  ", vi.runLen, vi.groupID)
	}

	// Vertex type str
	fmt.Fprintf(out, "\n                  ")
	for _, vi := range Xvtx {
		desc := buf[:0]
		for _, e := range vi.edges {
			desc = e.WriteStr(desc)
		}
		fmt.Fprintf(out, "  %s  ", desc)
	}

	for ci := int32(0); ci < Nv; ci++ {
		traces_ci := int64(0)
		for _, vi := range Xvtx {
			traces_ci += int64(vi.runLen) * vi.traces[ci].cycles
		}
		fmt.Fprintf(out, "\n   C%d:%8d| ", ci+1, traces_ci)
		for _, vi := range Xvtx {
			fmt.Fprintf(out, "%8d  ", vi.traces[ci].cycles)
		}
	}

}

func (X *graphState) Traces(numTraces int) Traces {
	if numTraces <= 0 {
		numTraces = int(X.vtxCount)
	}

	X.calcUpTo(int32(numTraces))
	return X.traces[:numTraces]
}

// AP Hash Function
// https://www.partow.net/programming/hashfunctions/#AvailableHashFunctions
func APHash64(buf []byte) uint64 {
	var hash uint64 = 0xaaaaaaaaaaaaaaaa
	for i, b := range buf {
		if (i & 1) == 0 {
			hash ^= ((hash << 7) ^ uint64(b) ^ (hash >> 3))
		} else {
			hash ^= (^((hash << 11) ^ uint64(b) ^ (hash >> 5)) + 1)
		}
	}
	return hash
}

func writeU64(buf []byte, val uint64) {
	buf[0] = byte((val) & 0xFF)
	buf[1] = byte((val >> 8) & 0xFF)
	buf[2] = byte((val >> 16) & 0xFF)
	buf[3] = byte((val >> 24) & 0xFF)
	buf[4] = byte((val >> 32) & 0xFF)
	buf[5] = byte((val >> 40) & 0xFF)
	buf[6] = byte((val >> 48) & 0xFF)
	buf[7] = byte((val >> 56) & 0xFF)
}

type splitMix struct {
	h1 uint64
	h2 uint64
}

func (mix *splitMix) Reset() {
	mix.h1 = 0xaaaaaaaaaaaaaaaa
	mix.h2 = 0
}

func (mix *splitMix) Write(val int64) {
	x := mix.h1 ^ mix.h2

	// https://github.com/skeeto/hash-prospector
	x1 := x
	x1 = x1 + 0x9e3779b97f4a7c15 + uint64(val)
	x1 ^= (x1 >> 30)
	x1 *= 0xbf58476d1ce4e5b9
	x1 ^= (x1 >> 27)
	x1 *= 0x94d049bb133111eb
	x1 ^= (x1 >> 31)
	mix.h1 = x1

	// https://gist.github.com/badboy/6267743
	x2 := x
	x2 = (^x2) + (x2 << 21) // x2 = (x2 << 21) - x2 - 1;
	x2 = x2 ^ (x2 >> 24)
	x2 = (x2 + (x2 << 3)) + (x2 << 8) // x2 * 265
	x2 = x2 ^ (x2 >> 14)
	x2 = (x2 + (x2 << 2)) + (x2 << 4) // x2 * 21
	x2 = x2 ^ (x2 >> 28)
	x2 = x2 + (x2 << 31)
	mix.h2 = x2
}

func (mix *splitMix) Sum() [16]byte {
	var hash [16]byte

	writeU64(hash[:8], mix.h1)
	writeU64(hash[8:16], mix.h2)

	return hash
}
