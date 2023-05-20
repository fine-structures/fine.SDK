package graph

import (
	"fmt"
	"io"
	"sort"
)

type ComputeVtx struct {
	VtxGroup

	// group *VtxGroup use instead of GroupID lookup?
	VtxID uint32
	Ci0   []int64
	Ci1   []int64
}

func (v *ComputeVtx) RemoveEdge(remove *VtxEdge) {
	for i, e := range v.Edges {
		if e == remove {
			v.Edges = append(v.Edges[:i], v.Edges[i+1:]...)
			return
		}
	}
	panic("edge not found")
}

type VtxGraphVM struct {
	VtxGraph

	traces   []int64
	edgePool []*VtxEdge
	calcBuf  []int64
	vtx      []*ComputeVtx // Vtx by VtxID (zero-based indexing)

}

// func (X *Graph) Reclaim() {
// 	if X != nil {
// 		graphPool.Put(X)
// 	}
// }

// var graphPool = sync.Pool{
// 	New: func() interface{} {
// 		return new(Graph)
// 	},
// }

// func (X *VtxGraphVM) Reset() {
// 	X.Status = VtxStatus_Invalid
// 	X.Edges = X.Edges[:0]
// 	X.triVtx = nil
// }

// Adds a VtxEdge to the vtx / group ID.
// If the named vtx does not exist, it is implicitly created.
func (X *VtxGraphVM) addEdgeToVtx(dst uint32, e *VtxEdge) {
	Nv := len(X.vtx)
	dstID := int(dst)

	// Resize vtx table as necessary
	if cap(X.vtx) < dstID {
		old := X.vtx
		X.vtx = make([]*ComputeVtx, dstID, 8+2*cap(X.vtx))
		copy(X.vtx, old)
	} else if len(X.vtx) < dstID {
		X.vtx = X.vtx[:dstID]
	}

	// Create vtx if necessary
	for i := Nv; i < dstID; i++ {
		v := X.vtx[i]
		if v == nil {
			v = &ComputeVtx{}
			v.Edges = make([]*VtxEdge, 0, 8)
			X.vtx[i] = v
		} else {
			v.Edges = v.Edges[:0]
		}

		v.VtxID = uint32(i + 1)
		v.GroupID = 0
		v.VtxCount = 1
		Nv++
	}

	// With a dst vtx in hand, add the edge
	dstVtx := X.vtx[dstID-1]
	dstVtx.Edges = append(dstVtx.Edges, e)
}

func (X *VtxGraphVM) Vtx() []*ComputeVtx {
	return X.vtx
}

func (X *VtxGraphVM) VtxCount() int {
	return len(X.vtx)
}

func (X *VtxGraphVM) ResetGraph() {
	X.Status = VtxStatus_Invalid
	//X.VtxToGroupID = X.VtxToGroupID[:0]
	// for _, v := range X.vtx {
	// 	for _, e := range v.Edges {
	// 		X.edgePool = append(X.edgePool, e)
	// 	}
	// }
	X.vtx = X.vtx[:0]
	X.Edges = X.Edges[:0]
	X.edgePool = X.edgePool[:0]
	X.traces = nil
	X.Status = VtxStatus_Invalid

	// if cap(X.vtx) < Nv {
	// 	X.vtx = make([]*ComputeVtx, Nv, 8 + edgePool2*cap(X.vtx))
	// } else if len(X.vtx) < dstID {
	// 	X.vtx = X.vtx[:dstID]
	// }

	// for i, v := range X.vtx {
	//     if vi == nil {
	//     	v = &ComputeVtx{
	//     	   Edges: make([]*VtxEdge, 0, 8),
	// 		}
	//     	X.vtx[i] = v
	//     } else {
	// 		v.Edges = v.Edges[:0]
	// 	}
	//     v.VtxID = uint32(i+1)
	// }

}

func (X *VtxGraphVM) newVtxEdge() *VtxEdge {
	Ne := len(X.edgePool) + 1

	if cap(X.edgePool) < Ne {
		old := X.edgePool
		X.edgePool = make([]*VtxEdge, Ne, 16+2*cap(X.edgePool))
		copy(X.edgePool, old)
	} else {
		X.edgePool = X.edgePool[:Ne]
	}

	e := X.edgePool[Ne-1]
	if e == nil {
		e = &VtxEdge{}
		X.edgePool[Ne-1] = e
	} else {
		*e = VtxEdge{}
		e.Cycles = e.Cycles[:0]
	}

	return e
}

// Adds an edge using one-based indexing.
func (X *VtxGraphVM) AddVtxEdge(
	numNeg, numPos int32,
	vi, vj uint32,
) error {

	if numNeg == 0 && numPos == 0 {
		return nil
	}

	if vi < 1 || vj < 1 {
		return ErrInvalidVtxID
	}

	// add a new flow edge for each "side" of the edge
	adding := 1
	if vi != vj {
		adding++
	}

	// Add edge "halves" (one for each vertex to reflect graph flow)
	for i := 0; i < adding; i++ {
		ei := X.newVtxEdge()
		ei.DstVtxID = vi
		ei.SrcVtxID = vj
		ei.CountNeg = numNeg
		ei.CountPos = numPos

		X.addEdgeToVtx(vi, ei)

		// Add the other flow edge to the other vertex
		if adding == 2 {
			vi, vj = vj, vi
		}
	}

	return nil
}

func (X *VtxGraphVM) Validate() error {
	var err error

	// Xv := X.Vtx()
	// Xe := X.Edges

	// // Data structure parody check: check total edges on vtx match aggregate edge count
	// if err == nil {
	// 	Nev := 0
	// 	for _, v := range Xv {
	// 		Nev += len(v.Edges)
	// 	}
	// 	if Nev != len(Xe) {
	// 		err = ErrBrokenEdges
	// 	}
	// }

	// WIP
	// Check 3 x Nv == 2 x Ne
	// if err == nil {
	// 	Nv := int32(len(Xv))
	// 	Ne2 := int32(0)
	// 	for _, e := range Xe {
	// 		Ne2 += (e.C1_Pos + e.C1_Neg) * 2
	// 		Ne2 += (e.E1_Pos + e.E1_Neg) * 1
	// 		Ne2 += (e.E2_Pos + e.E2_Neg) * 2
	// 		Ne2 += (e.E3_Pos + e.E3_Neg) * 3
	// 	}
	// 	if Ne2 != 3*Nv {
	// 		err = ErrViolates2x3
	// 	}
	// }

	if err == nil {
		if X.Status < VtxStatus_Validated {
			X.Status = VtxStatus_Validated
		}
		return nil
	}

	return err
}

/*
func (X *VtxGraphVM) Consolidate() {

	// First sort edges so that edges that can be consolidated will be sequential
	sort.Slice(X.Edges, func(i, j int) bool {
		return X.Edges[i].Ord() < X.Edges[j].Ord()
	})

	Xv := X.Vtx()

	// Now accumulate edges with matching characteristics
	// Note that doing so invalidates edge.SrcVtxID values, so lets zero them out for safety.
	// Work right to left as we overwrite the edge array in place
	{
		L := byte(0)
		Xe := X.Edges
		Ne := int32(len(Xe))
		numConsolidated := int32(0)
		for R := int32(1); R < Ne; R++ {
			eL := Xe[L]
			eR := Xe[R]
			match := eL.Ord() == eR.Ord()

			// If exact match, absorb R into L, otherwise advance L (old R becomes new L)
			if match {
				eL.C1_Pos += eR.C1_Pos
				eL.C1_Neg += eR.C1_Neg

				eL.E0_Count += eR.E0_Count

				eL.E1_Pos += eR.E1_Pos
				eL.E1_Neg += eR.E1_Neg

				eL.E2_Pos += eR.E2_Pos
				eL.E2_Neg += eR.E2_Neg

				eL.E3_Pos += eR.E3_Pos
				eL.E3_Neg += eR.E3_Neg

				Xv[eR.DstVtxID-1].RemoveEdge(eR)
				numConsolidated++
			} else {
				L++
				Xe[L], Xe[R] = Xe[R], Xe[L] // finalize R into a new L *and* preserve L target (as an allocation)
			}
		}
		Ne -= numConsolidated
		X.Edges = Xe[:Ne]
	}

	// Normalize edge order on each vertex
	for _, vi := range X.Vtx() {
		edges := vi.Edges
		sort.Slice(edges, func(i, j int) bool {
			return edges[i].Ord() < edges[j].Ord()
		})
	}

}
*/

// func (vg *VtxGroup) sameCycleGroup(vg2 *VtxGroup) bool {
// 	for ii := range gi.Cycles {
// 		ci_new := int64(gi.VtxCount) * gi.Cycles[ii] + int64(gj.VtxCount) * gj.Cycles[ii]
// 		remainder := ci_new % int64(gi.VtxCount + gj.VtxCount)
// 		if remainder != 0 {
// 			return false
// 		}
// 	}
// 	return true
// }

/*
func sameCycleGroup(gi, gj *VtxGroup) bool {
	for ii := range gi.Cycles {
		// ci_new := int64(gi.VtxCount) * gi.Cycles[ii] + int64(gj.VtxCount) * gj.Cycles[ii]
		// remainder := ci_new % int64(gi.VtxCount + gj.VtxCount)
		remainder := gi.Cycles[ii] - gj.Cycles[ii]
		if remainder != 0 {
			return false
		}
	}
	return true
}

func (vg *VtxGroup) absorbGroup(vg2 *VtxGroup) bool {
	if !sameCycleGroup(vg, vg2) {
		return false
	}
	vtxCount := vg.VtxCount + vg2.VtxCount
	for ii := range vg.Cycles {
		ci_new := int64(vg.VtxCount)*vg.Cycles[ii] + int64(vg2.VtxCount)*vg2.Cycles[ii]
		vg.Cycles[ii] = ci_new / int64(vtxCount)
	}
	vg.VtxCount = vtxCount
	return true
}


// inserts the given vtx group into an matching existing group or creates a new one
func (X *VtxGraphVM) addVtxToGroup(v *VtxGroup) {

	var dstGroup *VtxGroup
	for _, gi := range X.groups {
		if gi.absorbGroup(v) {
			dstGroup = gi
			break
		}
	}

	// If no matching group found, create a new one
	if dstGroup == nil {
		Ng := len(X.groups)
		Ng++

		if cap(X.groups) < Ng {
			old := X.groups
			X.groups = make([]*VtxGroup, Ng, 8+2*cap(X.groups))
			copy(X.groups, old)
		} else {
			X.groups = X.groups[:Ng]
		}

		gr := X.groups[Ng-1]
		if gr == nil {
			gr = &VtxGroup{}
			gr.Edges = make([]*VtxEdge, 0, 8)
			X.groups[Ng-1] = gr
		}
		gr.VtxCount = v.VtxCount
		gr.Edges = gr.Edges[:0]
		gr.Cycles = append(gr.Cycles[:0], v.Cycles...)
		gr.GroupID = uint32(Ng)
		dstGroup = gr
	}

	// bind the vertex to its group
	v.GroupID = dstGroup.GroupID

	//X.VtxToGroupID = append(X.VtxToGroupID, byte(Ng))

	// gr := X.groups[Ng-1]
	// for _, se := range v.Edges {
	// 	// TODO: use pooling
	// 	e := &VtxEdge{}
	// 	*e = *se

	// 	//if cap(gr.Edges) == len(gr.Edges) {
	// 	gr.Edges = append(gr.Edges, e)
	// }

}
*/

// func (X *VtxGraphVM) setVtxGroup(v *ComputeVtx, groupID uint32) {
// 	v.GroupID = groupID

// 	gi := int(groupID)
// 	for gi >= len(X.groups) {
// 		X.groups = append(X.groups, make([]*VtxGroup, 0, 8))
// 	}

// 	X.groups[gi] = append(X.groups[gi], v)
// }

// func (X *VtxGraphVM) forEveryVtxPair(fn func(vi, vj *ComputeVtx)) {
// 	Xv := X.Vtx()

// 	for i, vi := range Xv {
// 		for _, vj := range Xv[i+1:] {
// 			//j := j0 + i + 1
// 			fn(vi, vj)
// 		}
// 	}
// }

// Pre: assume edges are consolidated
// func (v *ComputeVtx) hasDoubleEdge() (VtoE2 *VtxEdge, E2toV *VtxEdge, V_adj int32) {
// 	// if len(v.Edges) != 2 {
// 	// 	return
// 	// }
// 	// if
// }

func (X *VtxGraphVM) Canonize() {

	/*
		// Do this so that we can more easily detect consolidation opportunities
		for _, v := range X.Vtx() {
			X.consolidateEdges(v)
		}
		// Look for and normalize A=vi-vj=B to A=(vi,vj)=B.
		// This forms vi and vj into a single group while preserving cycle signature.
		//
		//                                    vi
		//       A=vi-vj=B     ===>          /|\
		//                                 A-vj-B
		//
		//  ±A±A±vi + ±B±B±vj  ===>  ±A±B±vi + ±A±B±vj  (must preserve sign aggregates)
		//  sA0 A * sA1 A * svi * vi +
		//  sB0 B * sB1 B * svj * vj
		//                     ===> (sA0 + sA1) * A * (svi * vi) +
		//                          (sB0 + sB1) * B * (svj * vj)
		X.forEveryVtxPair(func(Vi, Vj *ComputeVtx) {

			e_iA, e_Ai, Vi_adj := Vi.hasDoubleEdge()
			if e_iA == nil {
				return
			}

			e_jB, e_Bj, Vj_adj := Vj.hasDoubleEdge()
			if e_jB == nil {
				return
			}

			if e_iA.DstVtxID != e_Ai.SrcVtxID || e_jB.DstVtxID != e_Bj.SrcVtxID {
				panic("inconsistent edge data")
			}

			// Vi and Vj must share a single edge
			if Vi_adj != Vj_adj {
				return
			}

			// Split A's edges to vj
			if e_iA.E2_Pos > 0 {
				e_iA.E2_Pos--
				e_Ai.E2_Pos--
				X.AddVtxEdge(0, 1, e_iA.SrcVtxID, Vj.VtxID)
			} else if e_iA.E2_Neg > 0 {
				e_iA.E2_Neg--
				e_Ai.E2_Neg--
				X.AddVtxEdge(1, 0, e_iA.SrcVtxID, Vj.VtxID)
			} else {
				panic("vi_eA should be a double edge")
			}

			// Split B's edges to vi
			if e_jB.E2_Pos > 0 {
				e_jB.E2_Pos--
				e_Bj.E2_Pos--
				X.AddVtxEdge(0, 1, e_jB.SrcVtxID, Vi.VtxID)
			} else if e_jB.E2_Neg > 0 {
				e_jB.E2_Neg--
				e_Bj.E2_Neg--
				X.AddVtxEdge(1, 0, e_jB.SrcVtxID, Vi.VtxID)
			} else {
				panic("e_jB should be a double edge")
			}

		})
	*/

	X.Traces(12)

	// Normalize edge signs -- duplicate every edge we find on a vtx
	{
		edges := X.Edges[:0]

		for _, v := range X.Vtx() {
			for _, src_e := range v.Edges {

				// Edge signs have been "baked" into the cycle signature that we are also going to sign-normalize, so also normalize counts.
				// Drop terms with a normalized count of zero.
				countPos := abs(src_e.CountPos - src_e.CountNeg)
				if countPos == 0 {
					continue
				}

				// Spit edges into even and odd
				{
					e := X.newVtxEdge()

					// Edge signs have been "baked" into the cycle signature that we are also going to sign-normalize, so also normalize counts
					e.CountNeg = 0
					e.CountPos = countPos
					e.Cycles = append(e.Cycles[:0], src_e.Cycles...)
					edges = append(edges, e)
				}
			}
		}
		X.Edges = edges
	}

	X.normalizeEdges()
}

func (X *VtxGraphVM) normalizeEdges() {

	edges := X.Edges

	// Sign and count normalization
	for _, e := range edges {
		sign := 0 // 0 means not yet determined
		for i, ci := range e.Cycles {

			// find first non-zero cycle and if possible factor out sign to get canonic form
			if sign == 0 && ci != 0 {
				if ci < 0 {
					sign = -1
					e.CountNeg, e.CountPos = e.CountPos, e.CountNeg
				} else {
					sign = 1
				}
			}
			if sign < 0 {
				e.Cycles[i] = -ci
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		ei := edges[i]
		ej := edges[j]

		// Then sort by cycle signature
		for i, ci := range ei.Cycles {
			d := ci - ej.Cycles[i]
			if d != 0 {
				return d < 0
			}
		}
		return false
	})

	// Consolidate edges (accumulate edges with compatible characteristics)
	// Note that doing so invalidates edge.SrcVtxID values, so lets zero them out for safety.
	// Work right to left as we overwrite the edge array in place
	consolidateEdges := true
	if consolidateEdges {
		L := 0
		Ne := len(edges)
		numConsolidated := 0
		for R := 1; R < Ne; R++ {
			eL := edges[L]
			eR := edges[R]

			match := true
			for i, ci := range eL.Cycles {
				if ci != eR.Cycles[i] {
					match = false
					break
				}
			}

			// If exact match, absorb R into L, otherwise advance L (old R becomes new L)
			if match {
				eL.CountNeg += eR.CountNeg
				eL.CountPos += eR.CountPos
				numConsolidated++
			} else {
				L++
				edges[L], edges[R] = edges[R], edges[L] // finalize R into a new L *and* preserve L target (as an allocation)
			}
		}
		Ne -= numConsolidated

		// Remove the all-zeros edge (can only be the first entry since they are sorted)
		if Ne > 0 {
			e0 := edges[0]
			zeros := true
			for _, ci := range e0.Cycles {
				if ci != 0 {
					zeros = false
					break
				}
			}
			if zeros {
				copy(edges[0:], edges[1:Ne])
				Ne--
				edges[Ne] = e0
			}
		}

		X.Edges = edges[:Ne]
	}

}

/*

func (X *VtxGraphVM) Normalize() {
	//X.Canonize() // TODO: check VtxStatus and skip when possible

	// Normalize group edge pairs into loops
	// Choice normalize edges to loops or loops to edges?
	//    - Loops to edges preserves particle count but can cause particlews to combine (e.g. Higgs factorization)
	//    - Edges to loops can cause a particle to be broken up (e.g. Higgs) -- but is prolly better for factorization
	//    -
	for _, gr := range X.groups {
		for _, e := range gr.Edges {

			if loopsToEdges := false; loopsToEdges {
				if e.SrcVtxID != e.DstVtxID {
					normE1 := min(e.E1_Pos, e.E1_Neg)
					if normE1 > 0 {
						e.E1_Pos -= normE1
						e.E1_Neg -= normE1

						e.C1_Pos += normE1
						e.C1_Neg += normE1
					}
				}
			}

			if edgesToLoops := true; edgesToLoops {
				// Safety assert -- remove in future
				if e.C1_Pos > 0 || e.C1_Neg > 0 {
					if e.SrcVtxID != e.DstVtxID {
						panic("loops must be self edges")
					}
				}

				normC1 := min(e.C1_Pos, e.C1_Neg)
				if normC1 > 0 {
					e.C1_Pos -= normC1
					e.C1_Neg -= normC1

					e.E1_Pos += normC1
					e.E1_Neg += normC1
				}
			}
		}
	}

	// Experiment: is is even possible to have a mult vtx group vtx and not have it be divisible by its count?:
	// If we can factor it out, simplifying the edge representation (allowing it to be represented by a symbol set -- e.g. 1, 4, 9)
	// This would also condense group vtx representation
	{
		tst := make([]int32, 16)

		for _, gr := range X.groups {
			if gr.VtxCount > 1 {
				for _, e := range gr.Edges {
					tst = tst[:0]

					tst = append(tst, e.E1_Pos+e.E1_Neg + e.C1_Pos+e.C1_Neg)
					tst = append(tst, e.E2_Pos+e.E2_Neg)

					for ti, t := range tst {
						if t%int32(gr.VtxCount) != 0 {
							fmt.Printf("tst[%d] == %d, gr.VtxCount = %d\n", ti, t, gr.VtxCount)
							//log.Fatalf("mult vtx group vtx not divisible by its count", ti, t, gr.VtxCount)
						}
					}
				}
			}
		}
	}

}
*/

func assert(cond bool, desc string) {
	if !cond {
		panic(desc)
	}
}

func (X *VtxGraphVM) Traces(numTraces int) Traces {
	if X.Status < VtxStatus_Validated {
		return nil
	}

	Nc := numTraces
	if Nc <= 0 {
		Nc = X.VtxCount()
	}

	if len(X.traces) < Nc {
		X.calcTracesTo(Nc)
	}
	return X.traces[:Nc]
}

func (X *VtxGraphVM) calcTracesTo(Nc int) {
	if Nc <= 0 {
		Nc = X.VtxCount()
	}

	X.setupBufs(Nc)

	Xv := X.Vtx()
	Nv := len(Xv)

	// Init edge (VM) state
	for i, vi := range Xv {
		for j := 0; j < Nv; j++ {
			vi.Ci0[j] = 0
		}
		vi.Ci0[i] = 1

		for _, e := range vi.Edges {
			for i := range e.Cycles {
				e.Cycles[i] = 0
			}
		}
	}

	// Oh Lord, our Adonai and God, you alone are the Lord. You have made the heavens, the heaven of heavens, with all their host, the earth and all that is on it, the seas and all that is in them; and you preserve all of them; and the host of heaven worships you. You are the Lord, the God, who chose Abram and brought him out of Ur of the Chaldeans and gave him the name Abraham; you found his heart faithful before you, and made with him the covenant to give the land of the Canaanites, the Hittites, the Amorites, the Perizzites, the Jebusites, and the Girgashites—to give it to his offspring. You have kept your promise, for you are righteous. And you saw the affliction of our fathers in Egypt and heard their cry at the Red Sea; and you performed signs and wonders against Pharaoh and all his servants and all the people of his land, for you knew that they acted arrogantly against them. And you made a name for yourself, as it is this day, and you divided the sea before them, so that they went through the midst of the sea on dry land, and you cast their pursuers into the depths, as a stone into mighty waters. Moreover in a pillar of cloud you led them by day, and in a pillar of fire by night, to light for them the way in which they should go. You came down also upon Mount Sinai, and spoke with them from heaven, and gave them right ordinances and true laws, good statutes and commandments; and you made known to them your holy sabbath, and commanded them commandments and statutes, a law for ever. And you gave them bread from heaven for their hunger, and brought forth water for them out of the rock for their thirst, and you told them to go in to possess the land that you had sworn to give them. But they and our fathers acted presumptuously and stiffened their neck, and did not obey your commandments. They refused to obey, neither were mindful of the wonders that you performed among them, but hardened their necks, and in their rebellion appointed a leader to return to their bondage. But you are a God ready to pardon, gracious and merciful, slow to anger, and abounding in steadfast love, and did not forsake them. Even when they had made for themselves a calf of molten metal, and~.
	// Yashhua is His name, Emmanuel, God with us!
	for ci := 0; ci < Nc; ci++ {
		odd := (ci & 1) == 0 // in zero-based indexing so odd indexes are even cycle indices.

		for _, vi := range Xv {

			// Alternate which is the prev / next state store
			Ci0, Ci1 := vi.Ci0, vi.Ci1
			if !odd {
				Ci0, Ci1 = Ci1, Ci0
			}

			for j, vj := range Xv {
				Ci1[j] = 0

				for _, e := range vj.Edges {
					assert(int(e.DstVtxID) == j+1, "edge DstVtxID mismatch")

					Ci_src := Ci0[e.SrcVtxID-1]
					netCount := int64(e.CountPos - e.CountNeg)
					Ci1[j] += netCount * Ci_src

					// Tally cycle returning to the home vtx on this vertex
					if int(vi.VtxID-1) == j {
						if e.CountNeg > e.CountPos {
							Ci_src = -Ci_src // negative weights negate cycle count
						}
						e.Cycles[ci] += Ci_src // store cycle components contributed by each edge.
					}
				}
			}

			vi_cycles_ci := Ci1[vi.VtxID-1]
			X.traces[ci] += vi_cycles_ci
			vi.Cycles[ci] = vi_cycles_ci
		}
	}
}

func (X *VtxGraphVM) PrintCycleSpectrum(numTraces int, out io.Writer) {
	TX := X.Traces(numTraces)

	//Xv := X.Vtx()
	Nc := len(TX)

	var buf [128]byte

	blank := "         "
	prOpts := PrintIntOpts{
		MinWidth: len(blank),
	}

	// Write header
	{
		line := buf[:0]
		line = append(line, "   DST  <=  SRC                       "...)

		for ti := range TX {
			ci := ti + 1
			if ci < 10 {
				line = append(line, ' ')
			}
			line = fmt.Appendf(line, "C%d      ", ti+1)
		}

		line = append(line, "\n  ---------------------------   "...)

		// append traces
		for _, Ti := range TX {
			line = AppendInt(line, Ti, prOpts)
		}

		line = append(line, '\n')
		out.Write(line)
	}

	for _, vi := range X.Vtx() {
		for _, ej := range vi.Edges {
			line := append(ej.AppendDesc(buf[:0]), "    "...)
			for _, c := range ej.Cycles[:Nc] {
				line = AppendInt(line, c, prOpts)
			}
			line = append(line, '\n')
			out.Write(line)
		}
	}

	out.Write([]byte(" -----===========---- \n"))

	{
		for _, ei := range X.Edges {
			line := ei.AppendDesc(buf[:0])
			line = append(line, "    "...)

			for i := 0; i < Nc; i++ {
				line = AppendInt(line, ei.Cycles[i], prOpts)
			}
			line = append(line, '\n')
			out.Write(line)
		}
	}

}

func (X *VtxGraphVM) setupBufs(Nc int) {
	Xv := X.Vtx()
	Nv := len(Xv)
	if Nc < Nv {
		Nc = Nv
	}

	need := Nc + Nv*(Nc+Nc+Nc)
	if len(X.calcBuf) < need {
		X.calcBuf = make([]int64, (need+15)&^15)
	}
	buf := X.calcBuf
	X.traces, buf = chopBuf(buf, Nc)
	for i := range X.traces {
		X.traces[i] = 0
	}

	// Place bufs on each vtx
	for _, v := range Xv {
		v.Ci0, buf = chopBuf(buf, Nc)
		v.Ci1, buf = chopBuf(buf, Nc)
		v.Cycles, buf = chopBuf(buf, Nc)

		for _, e := range v.Edges {
			if cap(e.Cycles) < Nc {
				e.Cycles = make([]int64, (need+3)&^3)
			}
			e.Cycles = e.Cycles[:Nc]
		}
	}

}

type GraphEncodingOpts int

func (X *VtxGraphVM) AppendGraphEncoding(io []byte, opts GraphEncodingOpts) []byte {
	X.Canonize()

	// Next steps:
	//   - use VtxStatus to prevent redundant work
	//   - encode canonized vtx edges (gr.Edges) as graph encoding, encoding signs LAST
	//   - at what point are graphs redundant and can be dropped?   for example:
	//       1. same canonization but different sign distribution (e.g. elastance of two Higgs 24+08)
	//             -- current view: counted as a dupe
	//       2. graphs with canonization synonyms (e.g. K8 with or without C1)
	//             -- "normalization" is to choose the canonic form
	//       3. graph canonized with dropped sign "deltas" (e.g. Higgs normalized to 24)
	//              -- requires a "normalization" step to choose the canonic form
	//              -- must prove that the canonic form is unique (1:1 mapping of existing Traces and this new form)

	// The key question to know more on is when (if) dupes need to be tallied (like Griggs said),
	//     or maybe they don't matter, can be dropped, and all the matters is the canonic form (plus state offset).
	// That is it of any importance there are 3 proton variants and 7 neutron variants, or if they can be dropped and only the canonic form matters?
	// Gut says the latter since who cares how many conceptual Griggs graphs are possible compared to what is the information needed to reproduce a given state.
	//    - Is "state" what graph is implied or is it the min information needed to reproduce a particles (traces) or a traces plus ????

	return nil
}

func abs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
