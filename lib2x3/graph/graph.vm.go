package graph

import (
	"errors"
	"io"
	"sort"
)

var (
	ErrBadEncoding   = errors.New("bad graph encoding")
	ErrBadVtxID      = errors.New("bad graph vertex ID")
	ErrMissingVtxID  = errors.New("missing vertex ID")
	ErrBadEdge       = errors.New("bad graph edge")
	ErrBadEdgeType   = errors.New("bad graph edge type")
	ErrBrokenEdges   = errors.New("bad or inconsistent graph edge configuration")
	ErrViolates2x3   = errors.New("graph is not a valid 2x3")
	ErrVtxExpected   = errors.New("vertex ID expected")
	ErrSitesExceeded = errors.New("number of loops and edges exceeds 3")
	ErrNilGraph      = errors.New("nil graph")
	ErrInvalidVtxID  = errors.New("invalid vertex or group ID")
)

func chopBuf(consume []int64, N int) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}

type ComputeVtx struct {
	// Initially assigned label: 1, 2, 3, ..  (a one-based index ID)
	VtxID uint32 `json:"VtxID,omitempty"`

	// GroupID assignment of home Vtx based on canonic cycle vector comparison ordering.
	// Init to 0 to denote unknown; the first is 1 (a one-based index ID)
	GroupID uint32 `json:"GroupID,omitempty"`

	// When assigned from a 2x3 graph, each Vtx has 3 edges but after canonicalization,
	// edges are pooled together into their respective groups.
	// Note: this array used one-based indexing, so the first element is always nil.
	Edges []*VtxEdge `json:"Edges,omitempty"`

	Cycles []int64 `json:"Cycles,omitempty"`
	Ci0    []int64 `json:"Ci0,omitempty"`
	Ci1    []int64 `json:"Ci1,omitempty"`
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

	calcBuf []int64
	vtx     []*ComputeVtx // Vtx by VtxID (zero-based indexing)
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

	if cap(X.vtx) < dstID {
		old := X.vtx
		X.vtx = make([]*ComputeVtx, dstID, 8+2*cap(X.vtx))
		copy(X.vtx, old)
	} else if len(X.vtx) < dstID {
		X.vtx = X.vtx[:dstID]
	}
	for i := Nv; i < dstID; i++ {
		v := X.vtx[i]
		if v == nil {
			v = &ComputeVtx{
				Edges: make([]*VtxEdge, 0, 8),
			}
			X.vtx[i] = v
		} else {
			v.Edges = v.Edges[:0]
		}
		v.VtxID = uint32(i + 1)
		v.GroupID = 0
		Nv++
	}

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
	X.Edges = X.Edges[:0]
	X.vtx = X.vtx[:0]
	X.Touch()

	// if cap(X.vtx) < Nv {
	// 	X.vtx = make([]*ComputeVtx, Nv, 8 + 2*cap(X.vtx))
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

func (X *VtxGraphVM) Touch() {
	X.Status = VtxStatus_Invalid
	X.Traces = nil
}

// Adds an edge using one-based indexing.
func (X *VtxGraphVM) AddVtxEdge(
	numNeg, numPos byte,
	vi, vj uint32,
) error {

	if numNeg+numPos == 0 {
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
	Ne := len(X.Edges) + adding

	if (cap(X.Edges) - Ne) < 1 {
		old := X.Edges
		X.Edges = make([]*VtxEdge, Ne, 16+2*cap(X.Edges))
		copy(X.Edges, old)
	} else {
		X.Edges = X.Edges[:Ne]
	}

	// Add edge "halves" (one for each vertex to reflect graph flow)
	for i := Ne - adding; i < Ne; i++ {
		ei := X.Edges[i]
		if ei == nil {
			ei = &VtxEdge{}
			X.Edges[i] = ei
		}
		*ei = VtxEdge{
			DstVtxID: vi,
			SrcVtxID: vj,
		}

		switch {
		case vi == vj:
			ei.C1_Pos = int32(numPos)
			ei.C1_Neg = int32(numNeg)

		case numPos == 1 && numNeg == 0:
			ei.E1_Pos = 1
		case numPos == 0 && numNeg == 1:
			ei.E1_Neg = 1

		case numPos == 2 && numNeg == 0:
			ei.E2_Pos = 1
		case numPos == 0 && numNeg == 2:
			ei.E2_Neg = 1
		case numPos == 1 && numNeg == 1:
			ei.E0_Count = 1

		case numPos == 3 && numNeg == 0:
			ei.E3_Pos = 1
		case numPos == 0 && numNeg == 3:
			ei.E3_Neg = 1
		case numPos == 2 && numNeg == 1:
			ei.E1_Pos = 1
		case numPos == 1 && numNeg == 2:
			ei.E1_Neg = 1

		default:
			return ErrBadEdgeType
		}

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

	Xv := X.Vtx()
	Xe := X.Edges

	// Data structure parody check: check total edges on vtx match aggregate edge count
	if err == nil {
		Nev := 0
		for _, v := range Xv {
			Nev += len(v.Edges)
		}
		if Nev != len(Xe) {
			err = ErrBrokenEdges
		}
	}

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

func (X *VtxGraphVM) Consolidate() {

	// First sort edges so that edges that can be consolidated will be sequential
	sort.Slice(X.Edges, func(i, j int) bool {
		return X.Edges[i].Ord() < X.Edges[j].Ord()
	})

	Xv := X.Vtx()

	// Now accumulate edges with matching characteristics
	// Note that doing so invalidates edge.srvVtx values, so lets zero them out for safety.
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

func assert(cond bool, desc string) {
	if !cond {
		panic(desc)
	}
}

func (X *VtxGraphVM) GetTraces(numTraces int) []int64 {
	Nc := numTraces
	if Nc <= 0 {
		Nc = X.VtxCount()
	}

	if X.Status < VtxStatus_Validated {
		return nil
	}

	if len(X.Traces) < Nc {
		X.calcTracesTo(Nc)
	}

	return X.Traces[:Nc]
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
	}

	// Oh Lord, our Adonai and God, you alone are the Lord. You have made the heavens, the heaven of heavens, with all their host, the earth and all that is on it, the seas and all that is in them; and you preserve all of them; and the host of heaven worships you. You are the Lord, the God, who chose Abram and brought him out of Ur of the Chaldeans and gave him the name Abraham; you found his heart faithful before you, and made with him the covenant to give the land of the Canaanites, the Hittites, the Amorites, the Perizzites, the Jebusites, and the Girgashites—to give it to his offspring. You have kept your promise, for you are righteous. And you saw the affliction of our fathers in Egypt and heard their cry at the Red Sea; and you performed signs and wonders against Pharaoh and all his servants and all the people of his land, for you knew that they acted arrogantly against them. And you made a name for yourself, as it is this day, and you divided the sea before them, so that they went through the midst of the sea on dry land, and you cast their pursuers into the depths, as a stone into mighty waters. Moreover in a pillar of cloud you led them by day, and in a pillar of fire by night, to light for them the way in which they should go. You came down also upon Mount Sinai, and spoke with them from heaven, and gave them right ordinances and true laws, good statutes and commandments; and you made known to them your holy sabbath, and commanded them commandments and statutes, a law for ever. And you gave them bread from heaven for their hunger, and brought forth water for them out of the rock for their thirst, and you told them to go in to possess the land that you had sworn to give them. But they and our fathers acted presumptuously and stiffened their neck, and did not obey your commandments. They refused to obey, neither were mindful of the wonders that you performed among them, but hardened their necks, and in their rebellion appointed a leader to return to their bondage. But you are a God ready to pardon, gracious and merciful, slow to anger, and abounding in steadfast love, and did not forsake them. Even when they had made for themselves a calf of molten metal, and~.
	// Yashhua is His name, Emmanuel, God with us!
	for ci := 0; ci < Nc; ci++ {
		odd := (ci & 1) != 0
		X.Traces[ci] = 0

		for _, vi := range Xv {

			// Alternate which is the prev / next state store
			Ci0, Ci1 := vi.Ci0, vi.Ci1
			if odd {
				Ci0, Ci1 = Ci1, Ci0
			}

			for j, vj := range Xv {
				Ci1[j] = 0
				groupEdgeWeight := int32(1)

				for _, e := range vj.Edges {
					assert(int(e.DstVtxID) == j+1, "edge DstVtxID mismatch")

					weight := int32(0)
					weight += 1 * (e.E1_Pos - e.E1_Neg)
					weight += 2 * (e.E2_Pos - e.E2_Neg)
					weight += 3 * (e.E3_Pos - e.E3_Neg)
					if e.SrcVtxID == e.DstVtxID {

						// Notice how every two cycles, C2 = C1
						groupEdgeWeight *= weight

						// Inject odd self weight
						weight = e.C1_Pos - e.C1_Neg

						// Inject even self weight (only on C2, C4, C6, ...)
						if odd {
							weight += groupEdgeWeight
							groupEdgeWeight = 1
						}
					}

					Cin := Ci0[e.SrcVtxID-1]
					Ci1[j] += int64(weight) * Cin
				}
			}

			vi_cycles_ci := Ci1[vi.VtxID-1]
			X.Traces[ci] += vi_cycles_ci
			vi.Cycles[ci] = vi_cycles_ci

			// // now add "even loop" generators
			// for j, vj := range Xv {
			// 	for _, e := range vj.Edges {
			// 		Cin := Ci0[e.SrcVtxID-1]
			// 		Ci1[j] += int64(e.EvenWeight) * Cin
			// 	}
			// }

		}
	}
}

func (X *VtxGraphVM) PrintCycleSpectrum(numTraces int, out io.Writer) {
	TX := X.GetTraces(numTraces)

	Xv := X.Vtx()
	Nc := len(TX)

	var buf [128]byte

	prOpts := PrintIntOpts{
		MinWidth: 9,
	}

	// Write header
	{
		line := buf[:0]
		line = append(line, "   DST  <=  SRC  x   C1     E0     E1      E2      E3           "...)

		for ti := range TX {
			line = append(line, 'C', byte(ti)+'1')
			line = append(line, "       "...)
		}

		line = append(line, "\n  -----------------------------------------------------  "...)

		// append traces
		for _, Ti := range TX {
			line = AppendInt(line, Ti, prOpts)
		}

		line = append(line, '\n')
		out.Write(line)
	}

	for _, vi := range Xv {
		for j, ej := range vi.Edges {
			line := ej.AppendDesc(buf[:0])
			if j == 0 {
				for _, c := range vi.Cycles[:Nc] {
					line = AppendInt(line, c, prOpts)
				}
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

	need := Nc + Nv*(Nv+Nv+Nc)
	if len(X.calcBuf) < need {
		X.calcBuf = make([]int64, (need+15)&^15)
	}
	buf := X.calcBuf
	X.Traces, buf = chopBuf(buf, Nc)

	// Place bufs on each vtx
	for _, v := range Xv {
		v.Ci0, buf = chopBuf(buf, Nv)
		v.Ci1, buf = chopBuf(buf, Nv)
		v.Cycles, buf = chopBuf(buf, Nc)
	}

}
