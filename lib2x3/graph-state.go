package lib2x3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/2x3systems/go2x3/lib2x3/graph"
)

func chopBuf(consume []int64, N int) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}

// GraphTriID is a 2x3 cycle spectrum encoding
type GraphTriID []byte

var (
	ErrNilGraph = errors.New("nil graph")
	ErrBadEdges = errors.New("edge count does not correspond to vertex count")
)

type graphState struct {
	vtxCount  int
	vtxDimSz  int
	vtxByID   []*graphVtx // vtx by initial vertex ID
	vtx       []*graphVtx // ordered list of vtx groups
	numGroups int32       // count of unique vertex groups present
	curCi     int
	traces    graph.Traces
	graph.VtxStatus
}

// triVtx starts as a vertex in a conventional "2x3" vertex+edge and used to derive a canonical LSM-friendly encoding (i.e. TriID)
// El Shaddai's Grace abounds.  Emmanuel, God with us!  Yashua has come and victory has been sealed!
type triVtx struct {
	graph.GroupID              // which group this is
	//VtxType                    // which type of vertex this is
	VtxIdx        byte         // Initial vertex ID (zero-based index)
	GroupT0       int8         // Traces sum of path length 1 (net sum of loops for group)
	edges         [3]groupEdge // Edges to other vertices
}

type groupEdge struct {
	FromVtxIdx   byte          // initial source vertex index (zero-based)
	FromGroup    graph.GroupID // the group ID of the vertex associated with FromVtx
	EdgeSign     int32         // -1, 0, +1; 0 denotes an edge normalized to 0
	EdgeSign_Raw int32         // -1  or +1
}

func (e groupEdge) Ord() int32 {
	ord := int32(e.FromGroup) << 2
	if e.EdgeSign > 0 {
		ord |= 0x01
	} else {
		ord |= 0x02
	}
	return ord
}

func (e groupEdge) EdgeTypeOrd(ascii bool) byte {
	r := byte('!')
	switch e.EdgeSign {
	case +1:
		r = ' '
	case -1:
		r = '_'
	case 0:
		r = '0'
	}
	return r
}

type graphVtx struct {
	triVtx
	cycles []int64 // for traces cycle fingerprint for cycles ci
	Ci0    []int64 // matrix row of X^i for this vtx -- by initial vertex ID
	Ci1    []int64 // matrix row of X^(i+1) for this vtx,
}

type EdgeTrait int

const (
	EdgeTrait_HomeVtxID EdgeTrait = iota
	//EdgeTrait_NormalizeSign
	EdgeTrait_EdgeSign
	EdgeTrait_FromGroup
	//EdgeTrait_EdgeType
	EdgeTrait_HomeGroup

	kNumLines = int(EdgeTrait_HomeGroup + 1)
)

func (v *graphVtx) AddLoop(from int, edgeSign int32) {
	v.AddEdge(from, edgeSign)
}

func (v *graphVtx) AddEdge(from int, edgeSign int32) {
	var ei int
	for ei = range v.edges {
		if v.edges[ei].EdgeSign == 0 {
			v.edges[ei] = groupEdge{
				FromVtxIdx:   byte(from),
				EdgeSign:     edgeSign,
				EdgeSign_Raw: edgeSign,
			}
			break
		}
	}

	if ei >= 3 {
		panic("tried to add more than 3 edges")
	}
}

func (v *graphVtx) Init(vtxIdx int) {
	v.VtxIdx = byte(vtxIdx)
	v.edges[0] = groupEdge{}
	v.edges[1] = groupEdge{}
	v.edges[2] = groupEdge{}
}

func (X *graphState) reset(Nv int) {

	X.vtxCount = Nv
	X.VtxStatus = graph.VtxStatus_Invalid
	X.Touch()
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

	for i := 0; i < Nv; i++ {
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

	Nv := Xsrc.NumVertices()
	X.reset(Nv)

	// Init vtx lookup map so we can find the group for a given initial vertex idx
	Xv := X.VtxByID()
	for i := 0; i < Nv; i++ {
		Xv[i].Init(i)
		X.vtx[i] = Xv[i]
	}

	// First, add edges that connect to the same vertex (loops)
	for i, vi := range Xv {
		vtype := Xsrc.vtx[i]
		for j := vtype.PosLoops(); j > 0; j-- {
			vi.AddLoop(i, +1)
		}
		for j := vtype.NegLoops(); j > 0; j-- {
			vi.AddLoop(i, -1)
		}
	}

	// Second, add edges connecting two different vertices
	for _, edge := range Xsrc.Edges() {
		ai, bi := edge.VtxIdx()
		pos, neg := edge.EdgeType().NumPosNeg()
		for j := pos; j > 0; j-- {
			Xv[ai].AddEdge(bi, +1)
			Xv[bi].AddEdge(ai, +1)
		}
		for j := neg; j > 0; j-- {
			Xv[ai].AddEdge(bi, -1)
			Xv[bi].AddEdge(ai, -1)
		}
	}

	// Calculate and assign siblings for every edge
	// This ensures we can sort (group) edges first by co-connectedness
	for _, v := range Xv {
		negLoops := byte(0)
		numEdges := byte(0)
		Ne := byte(0)
		for _, e := range v.edges {
			if e.EdgeSign != 0 {
				Ne++
				if e.FromVtxIdx == v.VtxIdx {
					if e.EdgeSign < 0 {
						negLoops++
					}
				} else {
					numEdges++
				}
			}
		}
		vtxType := GetVtxType(negLoops, numEdges)
		if Ne != 3 || vtxType == V_nil {
			return ErrBadEdges
		}
	}

	X.VtxStatus = graph.VtxStatus_Validated
	return nil
}

func (X *graphState) Vtx() []*graphVtx {
	return X.vtx[:X.vtxCount]
}

func (X *graphState) VtxByID() []*graphVtx {
	return X.vtxByID[:X.vtxCount]
}

func (X *graphState) sortVtxGroups() {
	Xv := X.Vtx()

	// With edges on vertices now canonic order, we now re-order to assert canonic order within each group.
	sort.Slice(Xv, func(i, j int) bool {
		vi := Xv[i]
		vj := Xv[j]

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
func (X *graphState) calcCyclesUpTo(numTraces int) {
	Nv := X.vtxCount

	if numTraces < Nv {
		numTraces = Nv
	}

	Xv := X.VtxByID()

	// Init C0
	if X.curCi == 0 {
		for i, vi := range Xv {
			for j := 0; j < Nv; j++ {
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

			for j := 0; j < Nv; j++ {
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

func (X *graphState) Touch() {
	if X.VtxStatus > graph.VtxStatus_Validated {
		X.VtxStatus = graph.VtxStatus_Validated
	}
	X.curCi = 0
	X.numGroups = 0
}

// func (X *graphState) forEveryGroupVtx(iter func(i int, vi *graphVtx)) {
// 	Nv := X.vtxCount
// 	Xv := X.Vtx()

// 	for i := int32(0); i < Nv; i++ {
// 		vi := Xv[i]
// 		for j := i + 1; j < Nv; j++ {
// 			vj := Xv[j]
// 			if vi.GroupID != vj.GroupID {
// 				iter(vi, vj)
// 			}
// 		}
// 	}
// }

func (X *graphState) forEveryNonGroupVtxPair(iter func(vi, vj *graphVtx)) {
	Nv := X.vtxCount
	Xv := X.Vtx()

	for i := 0; i < Nv; i++ {
		vi := Xv[i]
		for j := i + 1; j < Nv; j++ {
			vj := Xv[j]
			if vi.GroupID != vj.GroupID {
				iter(vi, vj)
			}
		}
	}
}

func (X *graphState) forEveryGroupEdgePair(iter func(vi, vj *graphVtx, ei, ej int)) {
	Nv := X.vtxCount
	Xv := X.Vtx()

	for i := 0; i < Nv; i++ {
		vi := Xv[i]
		for ei := 0; ei < 3; ei++ {
			for j := i; j < Nv; j++ {
				vj := Xv[j]
				if vi.GroupID != vj.GroupID {
					goto next_vi_edge
				}
				ej := 0
				if i == j {
					ej = ei + 1
				}
				for ; ej < 3; ej++ {
					iter(vi, vj, ei, ej)
				}
			}
		next_vi_edge:
		}
	}
}

func (X *graphState) Canonize() {

	for {
		if X.VtxStatus >= graph.VtxStatus_Canonized {
			return
		}

		Nv := X.vtxCount
		X.calcCyclesUpTo(Nv)

		Xv := X.Vtx()

		// Sort vertices by vertex's innate characteristics & cycle signature
		sort.Slice(Xv, func(i, j int) bool {
			vi := Xv[i]
			vj := Xv[j]

			// Sort by cycle count first and foremost
			// The cycle count vector (an integer sequence of size Nv) is what characterizes a vertex.
			for ci := 0; ci < Nv; ci++ {
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
				for ci := 0; ci < Nv; ci++ {
					if v.cycles[ci] != v_prev.cycles[ci] {
						X.numGroups++
						break
					}
				}
			}
			v.GroupID = graph.FormGroupID(X.numGroups)
			v_prev = v
		}

		// With cycle group numbers assigned to each vertex, assign srcGroup to all edges and then finally order edges on each vertex canonically.
		for _, v := range Xv {
			for ei, e := range v.edges {
				src_vi := X.vtxByID[e.FromVtxIdx]
				from := src_vi.GroupID
				switch {
				case src_vi.VtxIdx == v.VtxIdx:
					from = graph.GroupID_LoopVtx
				case src_vi.GroupID == v.GroupID:
					from = graph.GroupID_LoopGroup
				}
				v.edges[ei].FromGroup = from
			}
		}

		/*
			K8,000045,p=1,v=8,"2oBB 4OAC 2oBB 2_   6    ","2_   6    ","1^-2-3-6-7^-8-5-4-2 6-8 1-4",0,24,12,104,120,552,980,3400,
			      V       :    1         2    :    3         4         5         6    :    7         8    :
			  EDGE SIGN   :   _         _     :                                       :                   :
			  EDGE FROM   :   oBB       oBB   :   OAC       OAC       OAC       OAC   :   oBB       oBB   :
			    GROUP     :::AAAAAAAAAAAAAAA:::::BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:::::CCCCCCCCCCCCCCC:::

			becomes
			
			    V / G     :     1     :       2        :    3        
			  NRM COUNT   :   ?   ?   :   ?           :           :
			  NET COUNT   :  -2   4   :   4   4   4   :   2   4   :
			  ABS COUNT   :   2   4   :   4   4   4   :   2   4   :        :
			  EDGE FROM   :   O   B   :   O   A   C   :   O   B . :
			    GROUP     :::AAAAAAA:::::BBBBBBBBBBB:::::CCCCCCC:::
                 C1 		    -2            0             2
			  
			               
			 2,4 O   -> A  -2
			 4,4 B   -> A
			 
			 4,4 A   -> B
			 4,4,O   -> B   0
			 4,4 C . -> B
			
			 4,4,B . -> C
			 2,2 O . -> C  -2
			 
			 higgs. 
			 24,24 O -> A  0
             		
            photon
             6 6 .O  -> A  0
					
		
			// ALSO
			// Odd/Even edge normalization: 
			// vtx loops with opposite signs 'normalize' into group loops (retaining their sign)
			//  - this reflects traces sum of C1 sum not changing and then o -> O being identical after C1.
			// .  (a) opposite signs from any group (i.e. -1 +1) normalize to a group loop edge.
			// Cosnider:
			K8,000021,p=1,v=8,"2OBB 4OAC 2OBB 2_   6    ","2_   6    ","1-2-3-1-4~5-2 3-6-7-4 5-8-6 7-8",0,24,12,104,120,552,980,3400,
			      V       :    1         2    :    3         4         5         6    :    7         8    :
			  EDGE SIGN   :   _         _     :                                       :                   :
			  EDGE FROM   :   OBB       OBB   :   OAC       OAC       OAC       OAC   :   OBB       OBB   :
			    GROUP     :::AAAAAAAAAAAAAAA:::::BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:::::CCCCCCCCCCCCCCC:::
			    
			K8,000045,p=1,v=8,"2oBB 4OAC 2oBB 2_   6    ","2_   6    ","1^-2-3-6-7^-8-5-4-2 6-8 1-4",0,24,12,104,120,552,980,3400,
			      V       :    1         2    :    3         4         5         6    :    7         8    :
			  EDGE SIGN   :   _         _     :                                       :                   :
			  EDGE FROM   :   oBB       oBB   :   OAC       OAC       OAC       OAC   :   oBB       oBB   :
			    GROUP     :::AAAAAAAAAAAAAAA:::::BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB:::::CCCCCCCCCCCCCCC:::
			    
			X.forEveryGroup(func(gi []*graphVtx) {
				T0 := int8(0)
				for _, v := range gi {
					for _, e := range v.edges {
						if e.FromVtxIdx == v.VtxIdx {
							T0 += e.EdgeSign
						}
					}
				}
				for _, v := range gi {
					v.GroupT0 = T0
				}
			})


			// Group edge normalization (1): teo edges from the same group with opposite signs normalize to ^ (alias for 0).
				
			      V       :    1    :    2    :    3    :    4    :    5    :
			  EDGE SIGN   :         :         :         :         :         :
			  EDGE FROM   :   BBC   :   AAE   :   ADD   :   oCC   :   ooB   :
			    GROUP     :::AAAAA:::::BBBBB:::::CCCCC:::::DDDDD:::::EEEEE:::
			
			τ-  (tau)      ,000002,p=1,v=5," BBC  AAE  ADD  oCC  ooB         _ 2       _ ","        _ 2       _ ","1~2=3-4=5",3,25,27,165,243,
			      V       :    1    :    2    :    3    :    4    :    5    :
			  EDGE SIGN   :         :     _   :         :         :     _   :
			  EDGE FROM   :   BBC   :   AAE   :   ADD   :   oCC   :   ooB   :
			    GROUP     :::AAAAA:::::BBBBB:::::CCCCC:::::DDDDD:::::EEEEE:::
    
    
    
			      V       :    1    :    2         3    :
			  EDGE SIGN   :         :                   :
			  EDGE FROM   :   oBB   :   ooA       ooA   :
			    GROUP     :::AAAAA:::::BBBBBBBBBBBBBBB:::
			
			
			p+ (proton),000002,p=1,v=3," oBB 2ooA    _         _ ","   _         _ ","1~2-3","{{2,-1,0},{-1,1,1},{0,1,2}}",5,13,35,97,275,793,2315,6817,
			      V       :    1    :    2         3    :
			  EDGE SIGN   :     _   :               _   :
			  EDGE FROM   :   oBB   :   ooA       ooA   :
			    GROUP     :::AAAAA:::::BBBBBBBBBBBBBBB:::
			
			
			
			K4,000005,p=1,v=4," oBB 2OAC  oBB  _   3    "," _   3    ","1^-2-3-4-2 1-4",0,12,12,68,
			      V       :    1    :    2         3    :    4    :
			  EDGE SIGN   :   _     :                   :         :
			  EDGE FROM   :   oBB   :   OAC       OAC   :   oBB   :
			    GROUP     :::AAAAA:::::BBBBBBBBBBBBBBB:::::CCCCC:::
			    
			K4,000003,p=1,v=4," oBB  AAC  BDD  oCC  _ _   __  _        "," _ _   __  _        ","1^-~2~3=4",0,12,12,68,
			      V       :    1    :    2    :    3    :    4    :
			  EDGE SIGN   :   _ _   :    __   :   _     :         :
			  EDGE FROM   :   oBB   :   AAC   :   BDD   :   oCC   :
			    GROUP     :::AAAAA:::::BBBBB:::::CCCCC:::::DDDDD:::
			      
			X.forEveryGroupEdgePair(func(vi, vj *graphVtx, ei, ej int) {
				if vi.edges[ei].EdgeSign+vj.edges[ej].EdgeSign == 0 {
					src_ei_vi := X.vtxByID[vi.edges[ei].FromVtxIdx]
					src_ej_vj := X.vtxByID[vj.edges[ej].FromVtxIdx]
					if src_ei_vi.GroupID == src_ej_vj.GroupID {
						vi.edges[ei].EdgeSign = 0
						vj.edges[ej].EdgeSign = 0
					}
				}
			})

			// Group edge normalization (2): two negative edges from the same group flip positive
			X.forEveryGroupEdgePair(func(vi, vj *graphVtx, ei, ej int) {
				if vi.edges[ei].EdgeSign < 0 && vj.edges[ej].EdgeSign < 0 {
					src_ei_vi := X.vtxByID[vi.edges[ei].FromVtxIdx]
					src_ej_vj := X.vtxByID[vj.edges[ej].FromVtxIdx]
					if src_ei_vi.GroupID == src_ej_vj.GroupID &&
						src_ei_vi.GroupT0 == 0 &&
						src_ej_vj.GroupT0 == 0 {
						vi.edges[ei].EdgeSign = -vi.edges[ei].EdgeSign
						vj.edges[ej].EdgeSign = -vj.edges[ej].EdgeSign
					}
				}
			})
		*/

		// A=vi-vj=B => A-vi-B, A-vj-B
		//X.forEveryGroupVtxPair(func(vi, vj *graphVtx) {
		X.forEveryNonGroupVtxPair(func(vi, vj *graphVtx) {
			/*
				// Criterion #1(A): each vtx must have a double edge ( ±A±A + ±B±B => ±A±B + ±A±B)
				eA1, _, vA_adj := vA.hasDoubleGroupEdges()
				if vA_adj < 0 {
					continue
				}

				vB := X.vtxByID[vA.edges[vA_adj].srcVtx]

				// Criterion #2: the adjacent vtx must belong to a different group
				if vA.GroupID == vB.GroupID {
					continue
				}

				// Criterion #1(B)
				eB1, _, vB_adj := vB.hasDoubleGroupEdges()
				if vB_adj < 0 {
					continue
				}

				// Green light to rewire -- but then we must also trigger a re-canonicalize
				A1 := vA.edges[eA1]
				vA.edges[eA1] = vB.edges[eB1]
				vB.edges[eB1] = A1
				madeChanges = true
			*/
		})

		// 		func (v *triVtx) hasDoubleGroupEdges() (A1, A2, B int32) {
		// 	if e0, e1 := v.edges[0].GroupEdge, v.edges[1].GroupEdge; e0.GroupID() == e1.GroupID() {
		// 		return 0, 1, 2
		// 	}
		// 	if e0, e2 := v.edges[0].GroupEdge, v.edges[2].GroupEdge; e0.GroupID() == e2.GroupID() {
		// 		return 0, 2, 1
		// 	}
		// 	if e1, e2 := v.edges[1].GroupEdge, v.edges[2].GroupEdge; e1.GroupID() == e2.GroupID() {
		// 		return 1, 2, 0
		// 	}
		// 	return -1, -1, -1
		// }

		for _, v := range Xv {

			// With each edge srcGroup now assigned, we can order the edges canonically
			//
			// Canonically order edges by edge type then by edge sign & groupID
			// Note that we ignore edge grouping, which means graphs with double edges will encode onto graphs having a non-double edge equivalent.
			// For v=6, this turns out to be 4% less graph encodings (52384 vs 50664) and for v=8 about 2% less encodings (477k vs 467k).
			// If we figure out we need these encodings back (so they count as valid state modes), we can just export the grouping info.
			sort.Slice(v.edges[:], func(i, j int) bool {
				d := v.edges[i].Ord() - v.edges[j].Ord()
				return d < 0
			})
		}

		// if X.normalizeLoopsInGroups() {
		// 	X.Touch()
		// 	continue
		// }

		X.sortVtxGroups()

		X.VtxStatus = graph.VtxStatus_Canonized

	}
}

/*
func (X *graphState) normalizeLoopsInGroups() {
	Xv := X.Vtx()

	// We assume

	return false

	/*
		NEXT: break up "faux" groups -- vtx pairs that are in the same group and so can be made parallel (vs series)
		X=1-2=Y   => X-1-Y, X-2-Y,


		Xv := X.Vtx()

		// For each edge across a group boundary, see if an end has "inputs" that, when combined, are divisible by 2.
		// This implies the edges of these two vertices can be rewired into a (by definition) single cycle group of size 2:
		//  	     A=1-2=B => A-1-B, A-2-B
		//
		// In other words, look for two adjacent vertices that each have double edges to other groups.
		//
		// The more general basis here what can be changed while keeping the traces sum unchanged?
		// The answer is that flipping an edge sign in a group with |Gi| vertices means a corresponding number of negative terms can be absorbed via other sign flips.
		for _, vA := range Xv {

			// Criterion #1(A): each vtx must have a double edge ( ±A±A + ±B±B => ±A±B + ±A±B)
			eA1, _, vA_adj := vA.hasDoubleGroupEdges()
			if vA_adj < 0 {
				continue
			}

			vB := X.vtxByID[vA.edges[vA_adj].srcVtx]

			// Criterion #2: the adjacent vtx must belong to a different group
			if vA.GroupID == vB.GroupID {
				continue
			}

			// Criterion #1(B)
			eB1, _, vB_adj := vB.hasDoubleGroupEdges()
			if vB_adj < 0 {
				continue
			}

			// Green light to rewire -- but then we must also trigger a re-canonicalize
			A1 := vA.edges[eA1]
			vA.edges[eA1] = vB.edges[eB1]
			vB.edges[eB1] = A1
			madeChanges = true
		}

*/

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

       Family edges in          Family      Vtx type (ordinal enum)      Vtx Type
                                Counts      Counts                       Counts
ascii:
	3 +B+B-A  +A+C+B  +B+B+C    2 4 2       ||* ||| ||o                  2 4 2     --- ++- -++2 +++3


LSM:  (families ranked by family cardinality)
      edges from   counts  edge innate sign    count       edge type         edge sign modulate
	3 BBA ACB BBC  2 4 2   ++- +++  +++        2 4 2       ||* ||| ||o        --- ++- -++2 +++3

*/

func (X *graphState) getTraitRun(Xv []*graphVtx, vi int, trait EdgeTrait) int {
	Nv := len(Xv)
	var buf [8]byte

	viTr := Xv[vi].appendTrait(buf[:0], vi, trait, false)

	runLen := 1
	for vj := vi + 1; vj < Nv; vj++ {
		vjTr := Xv[vj].appendTrait(buf[4:4], vj, trait, false)
		if !bytes.Equal(viTr, vjTr) {
			break
		}
		runLen++
	}

	return runLen
}




func (X *graphState) ExportEncoding(io []byte, opts graph.ExportOpts) ([]byte, error) {
	X.Canonize()

	var buf [32]byte
	
	Xv := X.Vtx()
	Nv := len(Xv)
	ascii := (opts & graph.ExportAsAscii) != 0

	traits := make([]EdgeTrait, 0, 4)

	{
		traits = append(traits,
			EdgeTrait_FromGroup,
			//EdgeTrait_EdgeType,
			EdgeTrait_EdgeSign,
		)
	}

	for _, ti := range traits {
		runLen := 0
		RLE := buf[:0]
		for vi := 0; vi < Nv; vi += runLen {
			runLen = X.getTraitRun(Xv, vi, ti)

			// For readability, print the family count first in ascii mode (but for LSM it follows the edges)
			if ascii {
				if runLen == 1 {
					io = append(io, ' ')
				} else {
					io = strconv.AppendInt(io, int64(runLen), 10)
				}
			}

			io = Xv[vi].appendTrait(io, vi, ti, ascii)
			if ascii {
				io = append(io, ' ')
			} else {
				RLE = append(RLE, byte(runLen))
			}
		}

		io = append(io, RLE...)
	}

	return io, nil
}

func (v *triVtx) appendTrait(io []byte, vi int, trait EdgeTrait, ascii bool) []byte {
	switch trait {
	case EdgeTrait_HomeVtxID:
		vid := vi + 1
		d10 := byte(' ')
		if vid > 9 {
			d10 = '0' + byte(vid/10)
		}
		d01 := byte('0') + byte(vid%10)
		io = append(io, d10, d01, ' ')
	default:
		for _, e := range v.edges {
			r := byte('!')

			switch trait {
			case EdgeTrait_HomeGroup:
				r = v.GroupRune()
			case EdgeTrait_FromGroup:
				r = e.FromGroup.GroupRune()
			case EdgeTrait_EdgeSign:
				if e.EdgeSign != 0 {
					r = e.EdgeTypeOrd(ascii)
				} else {
					r = ' '
				}
			}
			io = append(io, r)
		}
	}
	return io
}

var lines = []EdgeTrait{
	EdgeTrait_HomeVtxID,
	EdgeTrait_EdgeSign,
	EdgeTrait_FromGroup,
	EdgeTrait_HomeGroup,
}

var lineLabels = []string{
	"      V       ",
	"  EDGE SIGN   ",
	"  EDGE FROM   ",
	"    GROUP     ",
}

// PrintVtxGrouping prints a graph in human readable form for the console.
// Note that this graph is not assumed to be in a canonic state.
//
// Example output for a neutron (canonic but not normalized):
//
//	    V       :    1    :    2         3    :    4    :
//	EDGE SIGN   :     _   :                   :     _   :
//	EDGE FROM   :   BBC   :   oOA       oOA   :   ooA   :
//	  GROUP     :::AAAAA:::::BBBBBBBBBBBBBBB:::::CCCCC:::
func (X *graphState) PrintVtxGrouping(out io.Writer) {
	X.Canonize()

	Nv := int(X.vtxCount)

	const vtxRad = 2
	vtxWid := 1 + (2 + (2*vtxRad + 1) + 2)
	totWid := (int(X.vtxCount) * vtxWid) + 1

	marginL := len(lineLabels[0])
	bytesPerRow := marginL + totWid + 1
	rows := make([]byte, kNumLines*bytesPerRow)
	for i := range rows {
		rows[i] = ' '
	}

	for i, li := range lines {
		row := rows[i*bytesPerRow:]
		copy(row, lineLabels[i])
		row = row[marginL:]

		switch li {
		// case kEdgeFrom:
		// 	xi := vtxWid / 2
		// 	for vi := 0; vi < Nv; vi++{
		// 		row[xi] = '.'
		// 		xi += vtxWid
		// 	}
		case EdgeTrait_HomeGroup:
			for xi := 0; xi < totWid; xi++ {
				row[xi] = ':'
			}
		}

		row[totWid] = '\n'
	}

	Xv := X.Vtx()

	var viEdges, vjEdges [3]byte

	for i, li := range lines {
		row := rows[i*bytesPerRow+marginL:]
		runL := 0

		for vi := 0; vi < Nv; {
			vtxRunLen := 1
			//traitConst := true

			Xv[vi].appendTrait(viEdges[:0], vi, EdgeTrait(li), true)

			for vj := vi + 1; vj < Nv; vj++ {
				if Xv[vi].GroupID != Xv[vj].GroupID {
					break
				}
				// Xv[vj].printEdgesDesc(vj, EdgeTrait(li), vjEdges[:])
				// if viEdges != vjEdges {
				// 	traitConst = false
				// }
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

			traitRun := false //traitConst && (li == kEdgeFrom || li == kEdgeType || li == kEdgeSign)

			for vj := vi; vj < vi+vtxRunLen; vj++ {
				vtxC := vtxWid*vj + vtxWid>>1

				if !traitRun {
					switch li {
					case EdgeTrait_HomeGroup:
					default:
						Xv[vj].appendTrait(vjEdges[:0], vj, EdgeTrait(li), true)
						copy(row[vtxC-1:], vjEdges[:])
					}
				}
			}

			switch li {
			case EdgeTrait_HomeGroup:
				c := Xv[vi].GroupRune()
				for w := runL + 3; w <= runR-3; w++ {
					row[w] = c
				}
			default:
				if traitRun {
					copy(row[runC-1:], viEdges[:])
				}
			}

			runL = runR
			vi += vtxRunLen
		}
	}

	out.Write(rows)
}

func (X *graphState) PrintCycleSpectrum(out io.Writer) {
	X.Canonize()

	Nv := X.vtxCount
	Xv := X.Vtx()

	for ci := 0; ci < Nv; ci++ {
		fmt.Fprintf(out, "%8d C%-2d", X.traces[ci], ci+1)
		for _, vi := range Xv {
			fmt.Fprintf(out, "%8d  ", vi.cycles[ci])
		}
		out.Write([]byte{'\n'})
	}
}

func (X *graphState) Traces(numTraces int) graph.Traces {
	numTraces = max(X.vtxCount, numTraces)
	X.calcCyclesUpTo(numTraces)
	return X.traces[:numTraces]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}