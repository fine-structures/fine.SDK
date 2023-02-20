package graph

import (
	"errors"
	"io"
	"sort"
)

var (
	ErrNilGraph = errors.New("nil graph")
	ErrInvalidVtxID = errors.New("invalid vertex or group ID")
)

func chopBuf(consume []int64, N int) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}

type ComputeVtx struct {

	// Initially assigned label: 1, 2, 3, ..  (a one-based index ID)
	// Group ID assignment of home Vtx based on canonic cycle vector comparison ordering.
	// Init to 0 to denote unknown; the first valid GroupID for a Vtx starts at GroupID_G1.
	VtxID  uint32 `protobuf:"varint,2,opt,name=GroupID,proto3" json:"GroupID,omitempty"`
	
	// When assigned from a 2x3 graph, each Vtx has 3 edges but after canonicalization,
	// edges are pooled together into their respective groups.
	// Note: this array used one-based indexing, so the first element is always nil.
	Edges []*VtxEdge `protobuf:"bytes,4,rep,name=Edges,proto3" json:"Edges,omitempty"`
	
	Cycles    []int64 `protobuf:"varint,10,rep,packed,name=Cycles,proto3" json:"Cycles,omitempty"`
	Cycles2   []int64 `protobuf:"varint,10,rep,packed,name=Cycles,proto3" json:"Cycles,omitempty"`

	Ci0    []int64 `protobuf:"varint,11,rep,packed,name=Ci0,proto3" json:"Ci0,omitempty"`
	Ci1    []int64 `protobuf:"varint,12,rep,packed,name=Ci1,proto3" json:"Ci1,omitempty"`
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
	tracesDimSz int // current dimSize for numTraces
	vtx []*ComputeVtx // Vtx by VtxID (zero-based indexing)
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
		X.vtx = make([]*ComputeVtx, dstID, 8 + 2*cap(X.vtx))
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
	    v.VtxID = uint32(i+1)
	    Nv++
	}
	
	dstVtx := X.vtx[dstID-1]
	dstVtx.Edges = append(dstVtx.Edges, e)	
}

func (X *VtxGraphVM) VtxCount() int {
	return len(X.vtx)
}

func (X *VtxGraphVM) ResetGraph() {
	X.Status = VtxStatus_Invalid
	X.Edges = X.Edges[:0]
	X.vtx = X.vtx[:0]
	
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


// Adds an edge using one-based indexing.
func (X *VtxGraphVM) AddVtxEdge(
	C1 int32, 
	vi, vj uint32, 
	edgeWeight int32,
) error {

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
		X.Edges = make([]*VtxEdge, Ne, 16 + 2*cap(X.Edges))
		copy(X.Edges, old)
	} else {
		X.Edges = X.Edges[:Ne]
	}
	
	// Add edge "halves" (one for each vertex to reflect graph flow)
	for i := Ne-adding; i < Ne; i++ {
		ei := X.Edges[i]
		if ei == nil {
			ei = &VtxEdge{}
			X.Edges[i] = ei
		}
		*ei = VtxEdge{
			C1: C1,
			DstVtxID: uint32(vi),
			SrcVtxID: uint32(vj),
		}
		if edgeWeight > 0 {
			ei.PosCount += edgeWeight
		} else if edgeWeight < 0 {
			ei.NegCount -= edgeWeight
		}
		
		X.addEdgeToVtx(vi, ei)
		
		// Add the other flow edge to the other vertex
		if adding == 2 {
			vi, vj = vj, vi
		}
	}
	
	return nil
}


func (X *VtxGraphVM) Validate2x3() error {


	


	return nil
	// TODO: validates the 2x3 graph
	// Check 3 x Nv == 2 x Ne, etc
}





func (X *VtxGraphVM) Canonize() {

	// First sort edges so that edges that can be consolidated will be sequential
	sort.Slice(X.Edges, func(i, j int) bool {
		if d := X.Edges[i].Ord() - X.Edges[j].Ord(); d != 0 {
			return d < 0
		}
		return false
	})

	// Now accumulate edges with matching characteristics 
	// Note that doing so invalidates edge.srvVtx values, so lets zero them out for safety.
	// Work right to left as we overwrite the edge array in place
	{
		L := byte(0)
		Xe:= X.Edges
		Ne := int32(len(Xe))
		//removed := make(map[*VtxEdge]struct{}, Ne)
		removed := int32(0)
		for R := int32(1); R < Ne; R++ {
			eL := Xe[L]
			eR := Xe[R]
			match := eL.Ord() == eR.Ord()

			// If exact match, absorb R into L, otherwise advance L (old R becomes new L)
			if match {
				eL.C1 += eR.C1
				eL.PosCount += eR.PosCount
				eL.NegCount += eR.NegCount
				X.vtx[eR.DstVtxID-1].RemoveEdge(eR)
				removed++
			} else {
				L++
				Xe[L], Xe[R] = Xe[R], Xe[L] // finalize R into a new L *and* preserve L target (as an allocation)
			}
		}
		Ne -= removed
		X.Edges = Xe[:Ne]
	}
	
	{
		Nev := 0
		for _, v := range X.vtx {
			Nev += len(v.Edges)
		}
		if Nev != len(X.Edges) {
			panic("vtx edge count does not match aggregate edge count")
		}
	}
	
	// Canonize of edges in each vtx
	for _, vi := range X.Vtx() {
		edges := vi.Edges
		sort.Slice(edges, func(i, j int) bool {
			return edges[i].Ord() < edges[j].Ord()
		})
	}
	
}

func (X *VtxGraphVM) Vtx() []*ComputeVtx {
	return X.vtx
}

func assert(cond bool, desc string) {
	if !cond {
		panic(desc)
	}
}

func (X *VtxGraphVM) PrintCycleSpectrum(numTraces int, out io.Writer) {
	X.setupBufs(numTraces)
	
	Xv := X.Vtx()
	Nv := len(Xv)
	
	// Init edge (VM) state
	for i, vi := range Xv {
		C1 := int64(0)
		for _, vie:= range vi.Edges {
			C1 += int64(vie.C1)
		}
		for j := 0; j < Nv; j++ {
			vi.Ci0[j] = 0
		}
		vi.Ci0[i] = C1
	}
	
	for _, e := range X.Edges {
		e.Ci1 = int64(e.C1Seed)
		e.PosCi0 = +e.Ci1 * int64(e.PosCount)
		e.NegCi0 = -e.Ci1 * int64(e.NegCount)
	}
	
	// Oh Lord, our Adonai and God, you alone are the Lord. You have made the heavens, the heaven of heavens, with all their host, the earth and all that is on it, the seas and all that is in them; and you preserve all of them; and the host of heaven worships you. You are the Lord, the God, who chose Abram and brought him out of Ur of the Chaldeans and gave him the name Abraham; you found his heart faithful before you, and made with him the covenant to give the land of the Canaanites, the Hittites, the Amorites, the Perizzites, the Jebusites, and the Girgashites—to give it to his offspring. You have kept your promise, for you are righteous. And you saw the affliction of our fathers in Egypt and heard their cry at the Red Sea; and you performed signs and wonders against Pharaoh and all his servants and all the people of his land, for you knew that they acted arrogantly against them. And you made a name for yourself, as it is this day, and you divided the sea before them, so that they went through the midst of the sea on dry land, and you cast their pursuers into the depths, as a stone into mighty waters. Moreover in a pillar of cloud you led them by day, and in a pillar of fire by night, to light for them the way in which they should go. You came down also upon Mount Sinai, and spoke with them from heaven, and gave them right ordinances and true laws, good statutes and commandments; and you made known to them your holy sabbath, and commanded them commandments and statutes, a law for ever. And you gave them bread from heaven for their hunger, and brought forth water for them out of the rock for their thirst, and you told them to go in to possess the land that you had sworn to give them. But they and our fathers acted presumptuously and stiffened their neck, and did not obey your commandments. They refused to obey, neither were mindful of the wonders that you performed among them, but hardened their necks, and in their rebellion appointed a leader to return to their bondage. But you are a God ready to pardon, gracious and merciful, slow to anger, and abounding in steadfast love, and did not forsake them. Even when they had made for themselves a calf of molten metal, and~.
	// Yashhua is His name, Emmanuel, God with us!
	for ci := 0; ci < numTraces; ci++ {
		odd := (ci & 1) != 0
		vi_traces_ci := int64(0)
		
		for _, vi := range Xv {
			
			// Alternate which is the prev / next state store
			Ci0, Ci1 := vi.Ci0, vi.Ci1
			if odd {
				Ci0, Ci1 = Ci1, Ci0
			}
			
			for j, vj := range Xv {
				Ci1[j] = 0
				for _, e := range vj.Edges {
					Cin := Ci0[e.SrcVtxID-1]
					assert(e.DstVtxID == j+1, "edge DstVtxID mismatch")
					netWeight := int64(e.PosCount - e.NegCount)
					if e.SrcVtxID == e.DstVtxID {
						groupEdgeWeight := int64(e.PosCount + e.NegCount)
						loopEdgeWeight := int64(e.C1)
						
						Ci1[j] += int64(e.C1) * Cin
					}
					Ci1[j] +=  netWeight * Cin
				}
			}

			vi_cycles_ci := Ci1[vi.VtxID-1]
			vi_traces_ci += vi_cycles_ci
			vi.Cycles = append(vi.Cycles, vi_cycles_ci)
		}
		X.Traces = append(X.Traces, vi_traces_ci)
		
	}

	// 		sum := int64(0)
	// 		for _, e := range vi.Edges {
	// 			e.Ci1 = int64(e.PosCount) * e.PosCi0 - e.NegCi0
	// 			e.Cout = e.Cin * int64(e.PosCount)
	// 			e.NegCi0 = -e.Cin * int64(e.NegCount)
				
	// 			// src := X.vtx[e.SrcVtxID].Ci0
	// 			// sum += 
	// 			// dot += int64(e.EdgeSign) * 
	// 		}
	// 		Ci1[j] = dot
			
			
	// 		// First stage: collect outputs from each edge and place into vtx history
	// 		for _, e := range vi.Edges {
				
			
			
	// 			line := e.AppendDesc(buf[:0]) 
	// 			line = append(line, '\n')
	// 			out.Write(line)
				
	// 			// if e.PosCount > 0 {
	// 			// 	fmt.Fprintf(out, "  %2d - %2d [label=%d];\n", i, e.DstVtxID, e.PosCount)
	// 			// }
	// 			// if e.NegCount > 0 {
	// 			// 	fmt.Fprintf(out, "  %d -> %d [label=%d];\n", i, e.DstVtxID, -e.NegCount)
	// 			// }
	// 		}
	// 	}
	// }
}


func (X *VtxGraphVM) setupBufs(numTraces int) {
	// dimSz := (numTraces+3) &^ 3

	// if cap(X.Traces) < int(numTraces) {
	// 	X.Traces = make([]int64, 0, dimSz)
	// } else {
	// 	X.Traces = X.Traces[:0]
	// }
	
	Xv := X.Vtx()
	Nv := len(Xv)

	need := numTraces + 3*Nv * numTraces
	if len(X.calcBuf) < need {
		X.calcBuf = make([]int64, (need + 15) &^ 15)
	}
	buf := X.calcBuf

	X.Traces, buf = chopBuf(buf, numTraces)
	
	// if X.Status > VtxStatus_Validated {
	// 	X.Status = VtxStatus_Validated
	// }
	// if X.tracesDimSz >= numTraces {
	// 	return
	// }

	// // Prevent rapid resize allocs
	// if Nv < 8 {
	// 	Nv = 8
	// }
	// X.vtxDimSz = Nv


	// Place cycle bufs on each vtx
	// buf := make([]int64, MaxVtxID+3*Nv*Nv)
	
	for _, v := range Xv {
		v.Ci0, buf = chopBuf(buf, Nv)
		v.Ci1, buf = chopBuf(buf, Nv)
		v.Cycles, buf = chopBuf(buf, Nv)
	}

}

/*
func (X *VtxGraphVM) calcCyclesUpTo() {

	Nv := X.vtxCount

	if numTraces < Nv {
		numTraces = Nv
	}

	Xv := X.VtxByID()

	// Init C0
	if X.curCi == 0 {
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
					dot += int64(e.EdgeSign) * Ci0[e.FromVtxIdx]
				}
				Ci1[j] = dot
			}

			vi_cycles_ci := Ci1[vi.VtxIdx]
			if ci < Nv {
				vi.cycles[ci] = vi_cycles_ci
			}
			traces_ci += vi_cycles_ci
		}
		X.traces[ci] = traces_ci
	}


}
*/