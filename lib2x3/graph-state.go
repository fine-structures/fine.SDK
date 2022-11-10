package lib2x3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
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
	tricoded
)

type graphState struct {
	vtxCount  int32
	vtxDimSz  int32
	vtxByID   []*graphVtx // vtx by initial vertex ID
	vtx       []*graphVtx // ordered list of vtx groups
	numGroups GroupID     // number of unique vertex groups present

	curCi  int32
	traces Traces
	triID  []byte
	status vtxStatus
	store  [256]byte
}

/*
type EdgeGrouping byte

const (
	Grouping_TBD        EdgeGrouping = 0 // Not yet known
	Grouping_111        EdgeGrouping = 1 // All 3 edges go to the same vertex
	Grouping_TripleEdge EdgeGrouping = 1 // All 3 edges go to the same vertex
	Grouping_112        EdgeGrouping = 2 // First 2 edges share the same vertex
	Grouping_DoubleEdge EdgeGrouping = 2 // First 2 edges share the same vertex
	Grouping_Disjoint   EdgeGrouping = 3 // Edges connect to different vertices

)

type VtxEdgesType int8
const (
	TriLoop VtxEdgesType = 0 // (e, ~e, π, ~π), 0 edges (always v=1)
	TwoLoop VtxEdgesType = 1 // "u" vertex; single edge
	OneLoop VtxEdgesType = 2 // "d" vertex; single loop
	NoLoops VtxEdgesType = 3 // "y" vertex, no loops, three edges
)

var VtxEdgesTypeDesc = []string{
	"***",
	"|**",
	"||*",
	"|||",
}
*/

// triVtx starts as a vertex in a conventional "2x3" vertex+edge and used to derive a canonical LSM-friendly encoding (i.e. TriID)
// El Shaddai's Grace abounds.  Emmanuel, God with us!  Yashua has come and victory has been sealed!
type triVtx struct {
	GroupID              // which group this is
	GroupSz byte         // Instances of this vtx type (signs and edges match exactly) ---- REMOVE??
	vtxID   byte         // Initial vertex ID (zero-based index)
	edges   [3]groupEdge // Edges to other vertices
}

type groupEdge struct {
	GroupEdge      // Baked sign, edge type, and source group ID (known after canonicalization via cycle spectrum sort)
	srcVtx    byte // initial source vertex index
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

func (v *triVtx) TriSign(forceEdgesPositive bool) TriSign {
	sign := byte(0)
	for _, e := range v.edges {
		sign <<= 1
		if e.sign < 0 {
			if e.isLoop || !forceEdgesPositive {
				sign |= 1
			}
		}
	}
	return TriSign(sign)
}

/*
// Returns sortable ordinal expressing the 3 bits in i={0,1,2} order:
//    Edges[0..2].IsLoop ? 0 : 1
func (v *triVtx) EdgesType() VtxEdgesType {
	edges := int32(3)
	for _, ei := range v.edges {
		if ei.isLoop {
			edges--
		}
	}
	return VtxEdgesType(edges)
}

func (v *triVtx) EdgesCycleOrd() int {
	ord := int(0)
	for _, ei := range v.edges {
		ord = (ord << 8) | int(ei.srcGroup)
	}
	return ord
}

func (v *triVtx) exactEdgesOrd() int {
	ord := 0
	for i := range v.edges {
		ord = (ord << 10) | int(v.sortEdgeOrd(i))
	}
	return ord
}

func (v *triVtx) familyEdgesOrd() int {
	ord := 0
	ord |= int(v.familyEdgeOrd(0)) << 20
	ord |= int(v.familyEdgeOrd(1)) << 10
	ord |= int(v.familyEdgeOrd(2))
	return ord
}
*/

// The purpose of the "family" edge reduction is to produce and edge mapping that funnels edges resulting in the same cycles ID to a the same edge edge code.
func (v *triVtx) familyEdgeOrd(ei int) uint8 {
	e := &v.edges[ei]
	cyclesID := e.GroupID()
	ord := uint8(cyclesID)
	if e.isLoop || cyclesID == v.GroupID {
		ord = 73 // 37 * 73 == 2701 Emmanuel!
		if e.isLoop && e.sign < 0 {
			ord--
		}
	}
	return ord
}

func (v *triVtx) sortEdgeOrd(ei int) int {
	cyclesID := v.edges[ei].GroupID()
	ord := int(cyclesID) << 1
	switch {
	case v.edges[ei].isLoop:
		ord |= 0x200
	case cyclesID == v.GroupID:
		ord |= 0x100
	}
	if v.edges[ei].sign > 0 {
		ord |= 0x1
	}
	return ord
}

func (v *triVtx) printEdges(tri []byte) {
	for ei, e := range v.edges {
		c := byte('?')
		if e.isLoop {
			c = 'o'
			if e.sign < 0 {
				c = '*'
			}
		} else if e.GroupID() == v.GroupID {
			c = '|'
		} else {
			c = e.GroupRune()
		}
		tri[ei] = c
	}
}

type VtxTrait int

const (
	VtxTrait_GroupID VtxTrait = iota
	VtxTrait_EdgesOrigin
	VtxTrait_InnateSign
	VtxTrait_EdgesType
	VtxTrait_ModulateSign
)

func (v *triVtx) appendTrait(io []byte, trait VtxTrait, ascii bool) []byte {
	switch trait {
	case VtxTrait_GroupID:
		io = append(io, v.GroupRune())
	case VtxTrait_EdgesOrigin:
		for _, e := range v.edges {
			io = append(io, e.GroupRune())
		}
	case VtxTrait_EdgesType: // Change to vtx symbol / type?
		if ascii {
			for _, e := range v.edges {
				io = append(io, e.EdgeTypeRune())
			}
		} else {
			ord := byte(0)
			for _, e := range v.edges {
				ord *= 3
				typ := byte(1)
				if e.isLoop {
					typ = 2
					if e.IsNeg() {
						typ = 0
					}
				}
				ord += typ
			}
			io = append(io, ord)
		}
	case VtxTrait_InnateSign, VtxTrait_ModulateSign:
		triSign := v.TriSign(trait == VtxTrait_InnateSign)
		if ascii {
			io = append(io, triSign.String()...)
		} else {
			io = append(io, byte(triSign))
		}
	}
	return io
}

// pre: v.edges[].cyclesID has been determined and set
func (v *triVtx) canonizeVtx() {

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
		return v.sortEdgeOrd(i) < v.sortEdgeOrd(j)
	})

	/*
		// At this point, the edges are canonic, so we can deterministically reorder any way we want.
		// So let's be classy and symmetrical and ensure o|o or |o|, etc.
		e0 := v.edges[0].LoopBit()
		e1 := v.edges[1].LoopBit()
		e2 := v.edges[2].LoopBit()
		switch {
		case (e1 == e2) && e0 != e1 && e0 != e2:
			v.edges[0], v.edges[1] = v.edges[1], v.edges[0]
		case (e0 == e1) && e2 != e1 && e2 != e0:
			v.edges[1], v.edges[2] = v.edges[2], v.edges[1]
		}
	*/

	/*
		// With edge order now canonic, determine the grouping.
		// Since we sort by edgeOrd, edges going to the same vertex are always consecutive and appear first
		// This means we only need to check a small number of cases
		if !v.edges[0].isLoop && v.edges[0].GroupID() == v.edges[1].GroupID() {
			if v.edges[1].GroupID() == v.edges[2].GroupID() {
				v.grouping = Grouping_111
			} else {
				v.grouping = Grouping_112
			}
		} else {
			v.grouping = Grouping_Disjoint
		}
	*/
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

	X.vtx = make([]*graphVtx, Nv, 2*Nv)
	X.vtxByID = X.vtx[Nv : 2*Nv]
	if cap(X.triID) == 0 {
		X.triID = X.store[:0]
	}

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

	// Count edges to see if we have a valid graph
	Ne := byte(0)

	// Calculate and assign siblings for every edge
	// This ensures we can sort (group) edges first by co-connectedness
	for _, v := range Xv {
		for _, ei := range v.edges {
			if ei.sign != 0 {
				Ne++
			}
		}
	}

	if Ne != 3*Nv {
		return ErrBadEdges
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

		// Larger groups appear first
		if d := int(vi.GroupSz) - int(vj.GroupSz); d != 0 {
			return d > 0
		}

		// Keep groups together
		if d := int(vi.GroupID) - int(vj.GroupID); d != 0 {
			return d < 0
		}

		// Sort by family edge only since we sort by sign below
		for e := range vi.edges {
			if d := int(vi.familyEdgeOrd(e)) - int(vj.familyEdgeOrd(e)); d != 0 {
				return d < 0
			}
		}

		// Then sort by sign
		for ei := range vi.edges {
			if d := vi.edges[ei].sign - vj.edges[ei].sign; d != 0 {
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

	Nv := X.vtxCount
	X.calcUpTo(Nv)

	Xg := X.Vtx()

	// Look for adjacent vtx that can be consolidated into an equivalent single cycle group
	//  e.g.    1^-~2     => 1^-2^,
	//          1^-~2-3=4 => 1^-2-3-4-2,1-4
	{
		// (1) Look for edge pairs on two adjacent vertices
		{

		}

		// (2) If remaining edge points to the other vtx, then these two vtx are the same cycle group since B+XX + A+YY <=> 2(A'XY)
		// This means the shared edge becomes a double edge in the new cycle group of these 2 vtx.
		{

		}
	}

	// Sort vertices by vertex's innate characteristics & cycle signature
	{
		sort.Slice(Xg, func(i, j int) bool {
			vi := Xg[i]
			vj := Xg[j]

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
	}

	// Now that vertices are sorted by cycle vector, assign each vertex the cycle group number now associated with its vertex index.
	{
		var groupSz [MaxVtxID]byte
		// var vtx [MaxVtxID]struct{
		// 	grpID GroupID
		// 	vi    byte
		// }

		X.numGroups = 1
		var v_prev *graphVtx
		//newGroup := true
		for _, v := range Xg {
			if v_prev != nil {
				for ci := int32(0); ci < Nv; ci++ {
					if v.cycles[ci] != v_prev.cycles[ci] {
						X.numGroups++
						break
					}
				}
			}
			// if X.numGroups == 0 || newGroup {
			// 	X.numGroups++
			// 	groups[X.numGroups].groupID = X.numGroups
			// }
			// vtx[vi].grpID = GroupID(X.numGroups)
			// vtx[vi].vi = byte(vi)
			v.GroupID = X.numGroups
			v_prev = v
			groupSz[v.GroupID]++
		}

		// Assign group rank from total count of each cycle group
		for _, v := range Xg {
			v.GroupSz = groupSz[v.GroupID]
		}

		// Resort by group rank
		sort.Slice(Xg, func(i, j int) bool {
			vi := Xg[i]
			vj := Xg[j]

			if d := int(vi.GroupSz) - int(vj.GroupSz); d != 0 {
				return d > 0
			}
			if d := int(vi.GroupID) - int(vj.GroupID); d != 0 {
				return d < 0
			}
			return false
		})

		// grID := GroupID(0)
		// prevID :=
		// for _, v := range Xg {
		// 	if v.GroupID != grID {

		// }

		// Reassign GroupIDs now that we are sorted by group cardinality
		//groupSz := 0
		grID := GroupID(0)
		for vi := 0; vi < len(Xg); {
			grID++
			grSz := int(Xg[vi].GroupSz)
			for j := 0; j < grSz; j++ {
				Xg[vi].GroupID = grID
				vi++
			}
		}

		// // With group cardinality known, remap group IDs based on size
		// {
		// 	sort.Slice(groups[:Nv], func(i, j int) bool {
		// 		return groups[i].groupSz > groups[j].groupSz
		// 	})
		// 	for vi, v := range Xg {
		// 		v.GroupID = groups[v.GroupID].groupID
		// 	}
		// }

		//for

	}

	//X.TriGraph.Canonize()

	/*
		tri := TriGroup{
			Grouping: v.grouping,
		}

		sign := TriSign(0)
		for i, ei := range v.edges {
			tri.Edges[i] = FormGroupEdge(GroupID(ei.srcGroup), ei.isLoop)
			sign <<= 1
			if ei.edgeSign < 0 {
				sign |= 1
			}
		}
		tri.Counts[sign] = int8(v.count)
		return tri
	*/

	Xv := X.VtxByID()

	// With cycle group numbers assigned to each vertex, assign srcGroup to all edges and then finally order edges on each vertex canonically.
	for _, v := range Xg {
		for ei, e := range v.edges {
			src_vi := Xv[e.srcVtx]
			v.edges[ei].GroupEdge = FormGroupEdge(src_vi.GroupID, e.isLoop, e.sign < 0)
		}

		// With each edge srcGroup now assigned, we can order the edges canonically
		v.canonizeVtx()
	}

	X.sortVtxGroups()

	// // reassign vertex IDs now that we're canonic
	// for vi, v := range Xg {
	// 	v.vtxID = byte(vi+1)
	// 	X.
	// }

	// {
	// 	// At this point, vertices are sorted via tricode (using group numbering via canonic ranking of cycle vectors)
	// 	// Here we collapse consecutive vertices with the same tricode into a "super" group
	// 	// As we collapse tricodes, we must reassign new groupIDs
	// 	X.TriGraph.Clear(len(Xg))
	// 	for i, vi := range Xg {
	// 		X.TriGraph.Groups[i] = vi.ExportTriGroup()
	// 	}
	// 	X.TriGraph.Consolidate()
	// }

	// Last but not least, we do an RLE-style compression of the now-canonic vertex series.
	// Note that doing so invalidates edge.srvVtx values, so lets zero them out for safety.
	// Work right to left
	/*
		{
			L := byte(0)
			for R := int32(1); R < Nv; R++ {
				XgL := Xg[L]
				XgR := Xg[R]
				identical := false
				if XgL.CyclesID == XgR.CyclesID && XgL.familyEdgesOrd() == XgR.familyEdgesOrd() {
					identical = true
					for ei := range XgL.edges {
						if XgL.edges[ei].srcGroup != XgR.edges[ei].srcGroup || XgL.edges[ei].sign != XgR.edges[ei].sign {
							identical = false
							break
						}
					}
				}

				// If exact match, absorb R into L, otherwise advance L (old R becomes new L)
				if identical {
					XgL.count += XgR.count
				} else {
					L++
					Xg[L], Xg[R] = Xg[R], Xg[L]
				}
			}
			X.numVtxGroups = L + 1
		}*/

	X.status = canonized
}

func (X *graphState) TriGraphExprStr() string {
	X.GraphTriID()

	var buf [GraphEncodingMaxSz]byte
	return string(X.AppendGraphEncoding(buf[:0], OutputAscii))
}

func (X *graphState) GraphTriID() GraphTriID {
	if X.status < tricoded {
		X.canonize()
		X.triID = X.AppendGraphEncoding(X.triID[:0], IncludeEdgeSigns)
		X.status = tricoded
	}

	return X.triID
}

// func (X *graphState) exportEncoding(io []byte, opts TriGraphEncoderOpts) (TriID []byte) {

// 	Xg := X.VtxGroups()

// 	// Gn - Vtx Family Group cardinality
// 	io = append(io, byte(len(Xg)))

// 	for _, g := range Xg {
// 		for ei := range g.edges {
// 			io = append(io, g.familyEdgeOrd(ei))
// 		}
// 	}

// 	for _, gi := range Xg {
// 		io = append(io, byte(gi.CyclesID))
// 	}

// 	for _, gi := range Xg {
// 		io = append(io, byte(gi.TriSign()))
// 	}

// 	for _, gi := range Xg {
// 		io = append(io, gi.count)
// 	}

// 	//io = append(io, []byte(X.CGE)...)
// 	return io
// }

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

type GraphEncodingOpts int32

const (
	OutputAscii GraphEncodingOpts = 1 << iota
	IncludeEdgeSigns

	___                = 1
	GraphEncodingMaxSz = (1 + ___ + 4*MaxVtxID + ___ + MaxVtxID + ___ + MaxVtxID*(3+___) + ___ + 8*(3+___) + 0xF) &^ 0xF
)

// func EncodingSzForTriGraph(Nv int32, opts GraphEncodingOpts) int32 {
// 	___ := 0
// 	if opts&OutputAscii != 0 {
// 		___ = 1
// 	}
// 	return (1 + ___ + Nv*(3 + ____) +  ___ + MaxVtxID + ___ + MaxVtxID*(3+___) + ___ + 8*(3+___)
// }

/*
	    Number of families
	    |   Family edges [*^ABCDEF..]
	    |   |   Family edges counts
	    |   |   |               Cycles composition
	    |   |   |               |           Edges sign modulation modes (8 counts)
		|   |   |               |           |
  UTF8:	2   *AC 2  *^B 2 ...    BB. AC.     N--- N--+ N-+- N-++ N+-- N+-+ N++- N+++
  Bits: 6   666 2  666 6        66  66      6    6    6    6    6    6    6    6

            Edge  Edge  Count  Edge
            From  Type         Sign
        2   AAC   *|o   1      +-+

      V       :     1           2     :     3           4           5           6     :     7           8     :
    FAMILY    ::::aaaaa:::::::aaaaa:::::::BBBBB:::::::BBBBB:::::::BBBBB:::::::BBBBB:::::::CCCCC:::::::CCCCC::::
    GROUP     ::::AAAAA:::::::AAAAA:::::::BBBBB:::::::BBBBB:::::::BBBBB:::::::BBBBB:::::::CCCCC:::::::CCCCC::::
  EDGE TYPE   :     .    ||*    .     :     .           .    |||    .           .     :     .    ||o    .     :
  EDGE FROM   :     .    BB     .     :     .           .    ACB    .           .     :     .    BB     .     :
  EDGE SIGN   :    ---         ++-    :    -++         -++         +++         +++    :     .    +++    .     :

  (for each cycle group)
    3 BBA ACB BBC 2 4 2   ||* ||| ||o 2 4 2     --- ++- -++2 +++3

  T = vtx types (16 enums, 4 bits)

  |||

  ||o
  ||*

  |**
  |*o
  |o*
  |oo

  ***   o**
  **o   o*o
  *o*   oo*
  *oo   ooo

       Family edges in          Family      Vtx type (ordinal enum)      Vtx Type
                                Counts      Counts                       Counts
ascii:
	3 +B+B-A  +A+C+B  +B+B+C    2 4 2       ||* ||| ||o                  2 4 2     --- ++- -++2 +++3


LSM:  (families ranked by family cardinality)
      edges from   counts  edge innate sign    count       edge type         edge sign modulate
	3 BBA ACB BBC  2 4 2   ++- +++  +++        2 4 2       ||* ||| ||o        --- ++- -++2 +++3

  maybe subsequently split runs by group *and* edge type ("group family")


   e-
   +A+A+A1 ooo1 +++1

   ~e+
   -A-A-A1 ***1 ---1



   higgs
   +A+A+A8 |||8 +++8
   +A+A+A8 |||8 ---8
   +A+A+A8 |||8 ---4 --+3 +++1


   photon
   +A+A+A1 |||2 +++2
   +A+A+A1 |||2 ---2

   tetra
   +A+A+A4 |||4 +++4

   sign RLE:
   count + sign ord
    5 bits   3 bits



*/

func (X *graphState) getTraitRun(Xv []*graphVtx, vi int, trait VtxTrait) int { //(triID []byte, modes []byte) {
	Nv := len(Xv)

	var buf [8]byte

	runLen := 1

	viTr := Xv[vi].appendTrait(buf[:0], trait, false)

	for vj := vi + 1; vj < Nv; vj++ {
		vjTr := Xv[vj].appendTrait(buf[4:4], trait, false)
		if !bytes.Equal(viTr, vjTr) {
			break
		}
		runLen++
	}

	return runLen
}

/*

	for vi := 0; vi < Nv; vi += runLen {
		io = Xv[vi].appendTrait(io, trait, ascii)
		runLen = 1
		for vj := vi + 1; vj < Nv; vj++ {
			if Xv[vi].CyclesID != Xv[vj].CyclesID {
				break
			}

			same := false
			vjTri = Xv[vj].appendTrait(vjTri[:0], trait, ascii)
			if

			runLen++
		}
	}

}
*/

func (X *graphState) AppendGraphEncoding(io []byte, opts GraphEncodingOpts) []byte { //(triID []byte, modes []byte) {
	// Generally, there's plenty of room (even accounting for adding spaces in ascii mode)
	// enc := io[len(io):cap(io)]
	// if len(enc) < GraphEncodingMaxSz {
	// 	enc = make([]byte, len(io) + GraphEncodingMaxSz)
	// 	copy(enc, io)
	// }

	var buf [32]byte

	Xv := X.Vtx()
	Nv := len(Xv)
	ascii := opts&OutputAscii != 0

	for ti := VtxTrait_EdgesOrigin; ti <= VtxTrait_ModulateSign; ti++ {
		runLen := 0
		RLE := buf[:0]
		for vi := 0; vi < Nv; vi += runLen {
			runLen = X.getTraitRun(Xv, vi, ti)
			io = Xv[vi].appendTrait(io, ti, ascii)
			if ascii {
				io = append(io, ' ')
				RLE = strconv.AppendInt(RLE, int64(runLen), 10)
				RLE = append(RLE, ' ')
			} else {
				RLE = append(RLE, byte(runLen))
			}
		}

		io = append(io, RLE...)
	}

	return io
}

/*
func (X *graphState) AppendGraphEncoding(io []byte, opts GraphEncodingOpts) []byte { //(triID []byte, modes []byte) {
	// Generally, there's plenty of room (even accounting for adding spaces in ascii mode)

	// enc := io[len(io):cap(io)]
	// if len(enc) < GraphEncodingMaxSz {
	// 	enc = make([]byte, len(io) + GraphEncodingMaxSz)
	// 	copy(enc, io)
	// }

	ascii := opts&OutputAscii != 0

	var runsBuf [MaxVtxID]byte
	Xruns := X.findGroupRuns(runsBuf[:0])

	Xg := X.VtxGroups()
	Nv := len(Xg)

	//
	// 	3 BBA ACB BBC   ++- +++  +++        2 4 2       ||* ||| ||o        --- ++- -++2 +++3
	//

	runLen := 0
	for vi := 0; vi < Nv; vi += runLen {
		runLen = 1
		for vj := vi + 1; vj < Nv; vj++ {
			if Xg[vi].GroupID != Xg[vj].GroupID {
				break
			}
			runLen++
		}
	}
	// c := byte(len(Xg))
	// if ascii {
	// 	c += '0'
	// }
	// io = append(io, c)
	// if ascii {
	// 	io = append(io, ':')
	// }

	var giTri [3]byte



	// // EDGE FROM GROUP
	// gi := byte(0)
	// for _, runLen := range Xruns {
	// 	g := Xg[gi]
	// 	c := byte(g.CyclesID)
	// 	if ascii {
	// 		c += 'A' - 1
	// 	}
	// 	io = append(io, c)

	// 	if ascii {
	// 		io = append(io, g.CyclesID.GroupRune())
	// 	} else {
	// 		io = append(io, g.CyclesID)
	// 	}
	// }

	// EDGE FAMILY GROUPS
	gi := byte(0)
	for _, runLen := range Xruns {
		g := Xg[gi]

		// For readability, print the family count first in ascii mode (but for LSM it follows the edges)
		c := byte(runLen)
		if ascii {
			if c == 1 {
				//io = append(io, ' ')
			} else {
				io = strconv.AppendInt(io, int64(runLen), 10)
			}
		}

		// TRI-CODE
		if ascii {
			g.printEdges(giTri[:])
		} else {
			giTri[0] = g.familyEdgeOrd(0)
			giTri[1] = g.familyEdgeOrd(1)
			giTri[2] = g.familyEdgeOrd(2)
		}
		io = append(io, giTri[:]...)
		if ascii {
			io = append(io, ' ')
		}

		if !ascii {
			io = append(io, runLen)
		}

		gi += runLen
	}

	// CYCLES COMPOSITION
	gi = 0
	for _, runLen := range Xruns {
		for j := byte(0); j < runLen; j++ {
			g := Xg[gi+j]
			io = g.appendTrait(io, VtxTrait_EdgesOrigin, ascii)
		}
		if ascii {
			io = append(io, ' ')
		}

		gi += runLen
	}

	// EDGES SIGNS
	gi = 0
	for _, runLen := range Xruns {
		for j := byte(0); j < runLen; j++ {
			g := Xg[gi+j]

			io = g.appendTrait(io, VtxTrait_ModulateSign, ascii)
			if ascii {
				io = append(io, ' ')
			}
		}
		if ascii {
			io = append(io, ' ', ' ')
		}

		gi += runLen
	}

	return io
}
*/

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
				r = e.SignRune()
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
					//groupChange = true
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

	// buf := [MaxVtxID]byte{}
	// Xruns := X.findVtxRuns(kEdgeHomeGroup, buf[:0])

	// vi := byte(0)
	// vtxIdx := 0
	// for _, runLen := range Xruns {
	// 	for j := byte(0); j < runLen; j++ {
	// 		g := Xg[gi+j]

	// 		vtxRunCount := int(runLen * g.count)
	// 		cen := margin + (dWidth * vtxIdx) / (vtxRunCount + 1)

	// 		for k := int(g.count); k > 0; k-- {

	// 			// Edge signs
	// 			for ei, e := range g.edges {
	// 				s := byte('+')
	// 				if e.sign < 0 {
	// 					s = '-'
	// 				}
	// 				rowS[cen + ei - 1] = s
	// 			}

	// 			// Cycle Group IDs
	// 			groupColor := byte('A' - 1 + g.CyclesID)
	// 			for w := -2; w <= 2; w++ {
	// 				rowC[cen + w] = groupColor
	// 			}

	// 			// Vertex IDs
	// 			ID := vtxIdx
	// 			if ID >= 10 {
	// 				rowV[cen - 1] = byte('0' + ID / 10)
	// 			}
	// 			rowV[cen] = '0' + byte(ID % 10)

	// 			vtxIdx++
	// 		}

	// 	}

	// 	gi += runLen
	// }

	// buf := [MaxVtxID]byte{}
	// Xruns := X.findGroupRuns(buf[:0])
	/*

		{
			rowS := rows[2 * bytesPerRow:]
			rowC := rows[3 * bytesPerRow:]
			rowV := rows[4 * bytesPerRow:]


			gi := byte(0)
			vtxIdx := 0
			for _, runLen := range Xruns {
				for j := byte(0); j < runLen; j++ {
					g := Xg[gi+j]

					vtxRunCount := int(runLen * g.count)
					cen := margin + (dWidth * vtxIdx) / (vtxRunCount + 1)

					for k := int(g.count); k > 0; k-- {

						// Edge signs
						for ei, e := range g.edges {
							s := byte('+')
							if e.sign < 0 {
								s = '-'
							}
							rowS[cen + ei - 1] = s
						}

						// Cycle Group IDs
						groupColor := byte('A' - 1 + g.CyclesID)
						for w := -2; w <= 2; w++ {
							rowC[cen + w] = groupColor
						}

						// Vertex IDs
						ID := vtxIdx
						if ID >= 10 {
							rowV[cen - 1] = byte('0' + ID / 10)
						}
						rowV[cen] = '0' + byte(ID % 10)

						vtxIdx++
					}


				}

				gi += runLen
			}
		}

		{

			rowS := rows[kEdgeFrom * bytesPerRow:]
			rowT := rows[1 * bytesPerRow:]

			gi := 0
			pos := margin

			for ri := 0; ri <= len(Xruns); ri++ {

				// Vertical separator line
				for yi := 0; yi < 3; yi++ {
					rows[pos + yi*bytesPerRow] = ':'
				}

				runLen := 0
				if ri < len(Xruns) {
					runLen = int(Xruns[ri])
				}

				runWidth := (dWidth * runLen) / len(Xg)
				for j := 0; j < runLen; j++ {
					g := Xg[gi+j]
					runC := pos + (runWidth * int(j) + runLen >>1) / runLen

					// Edge source group IDs
					for ei, e := range g.edges {
						edgeType := byte('|')
						if e.isLoop {
							edgeType = 'o'
							if e.sign < 0 {
								edgeType = '*'
							}
						}
						rowT[runC + ei] = edgeType
						if !e.isLoop {
							rowS[runC + ei] = byte('A' - 1 + e.srcGroup)
						}

						// else if e.srcGroup == v.CyclesID {
						// 	c = '|'
						// } else {
						// 	c = 'A' + byte(e.srcGroup) - 1
						// }
						// tri[ei] = c
					}


				}

				pos += runWidth
				gi += runLen
			}
		}

	*/

}

/*
func (X *graphState) PrintTriCodes(out io.Writer) {
	X.canonize()

	Xg := X.VtxGroups()

	var buf [256]byte
	var giTri, gjTri [3]byte

	// Vertex type str
	for line := 0; line < 4; line++ {
		fmt.Fprintf(out, "\n   %s  ", []string{
			"FAMILY EDGES ",
			"CYCLES ID    ",
			"SIGNS        ",
			"COUNT        ",
		}[line])

		for gi := 0; gi < len(Xg); {
			g := Xg[gi]

			// Calc col width based on grouping
			groupCols := 1
			g.printEdges(giTri[:])
			for gj := gi + 1; gj < len(Xg); gj++ {
				Xg[gj].printEdges(gjTri[:])
				if gjTri != giTri {
					break
				}
				groupCols++
			}
			col_ := buf[:3*groupCols+5*groupCols]
			col := col_[:len(col_)-5]
			for i := range col_ {
				col_[i] = ' '
			}

			c := byte('?')

			switch line {

			//case 0: // FAMILY ID
			// case 1: // EDGES  MASK
			// 	for gi := 0; gi < groupCols; gi++ {
			// 		v = Xg[vi+gi]
			// 		for ii, e := range v.edges {
			// 			if e.isLoop || e.srcGroup == v.CyclesID {
			// 				c = 'o'
			// 			} else {
			// 				c = 'A' + byte(e.srcGroup) - 1
			// 			}
			// 			col[gi*8+ii] = c
			// 		}
			// 	}
			case 0: // FAMILY EDGES
				center := (len(col) - 3) / 2
				copy(col[center:], giTri[:])

			case 1: // CYCLES ID
				for i := range col {
					col[i] = ':'
				}
				{
					for j := 0; j < groupCols; j++ {
						g = Xg[gi+j]
						c = 'A' + byte(g.CyclesID) - 1
						for k := 0; k < 3; k++ {
							col[j*8+k] = c
						}
					}
				}

			case 2: // SIGNS
				for j := 0; j < groupCols; j++ {
					g = Xg[gi+j]

					for ei, e := range g.edges {
						if e.sign < 0 {
							c = '-'
						} else {
							c = '+'
						}
						col[j*8+ei] = c
					}
				}

			case 3: // COUNT
				for j := 0; j < groupCols; j++ {
					g = Xg[gi+j]

					NNN := 1 //g.count
					for k := 2; k >= 0; k-- {
						col[j*8+k] = '0' + byte(NNN%10)
						NNN /= 10
					}
				}

			}
			out.Write(col_)
			gi += groupCols
		}
	}

}*/

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

	X.calcUpTo(int32(numTraces))
	return X.traces[:numTraces]
}
