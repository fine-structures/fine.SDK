package graph

import (
	fmt "fmt"
	io "io"
	"sort"
	"sync"

	"github.com/art-media-platform/amp.SDK/stdlib/tag"
	"github.com/fine-structures/fine.SDK/go2x3"
)

func NewState(src *state) *state {
	X := graphPool.Get().(*state)

	Nv := 0
	if src != nil {
		X.GraphID = src.GraphID
		X.traces = append(X.traces[:0], src.traces...)
		Nv = len(src.vtx)
	} else {
		X.GraphID = tag.ID{}
		X.traces = X.traces[:0]
	}

	if cap(X.vtx) >= Nv {
		X.vtx = X.vtx[:Nv]
	} else {
		for len(X.vtx) < Nv {
			X.vtx = append(X.vtx, VertexGroup{})
		}
	}

	for i := range X.vtx {
		v := &X.vtx[i]
		v.CopyFrom(&src.vtx[i])
	}
	return X
}

func (v *VertexGroup) Clear() {
	v.CopyFrom(nil)
}

func (v *VertexGroup) CopyFrom(v_in *VertexGroup) {
	if v_in == nil {
		*v = VertexGroup{
			Edges:  v.Edges[:0],
			Cycles: v.Cycles[:0],
			Ci:     v.Ci[:0],
		}
		return
	}

	*v = VertexGroup{
		VertexID:    v_in.VertexID,
		GroupID:     v_in.GroupID,
		CycleRadius: v_in.CycleRadius,
		VertexCount: v_in.VertexCount,
		Cycles:      v.Cycles[:0],
		Ci:          v.Ci[:0],
	}
}

var graphPool = sync.Pool{
	New: func() any {
		return &state{
			vtx:    make([]VertexGroup, 0, 24),
			traces: make([]int64, 0, 24),
		}
	},
}

type edgeSlot struct {
	VertexID int32
	EdgeID   int32
}

type state struct {
	GraphID     tag.ID
	cycleCount  int64         // cycle count, 0 denotes nil
	vertexCount int           // vertex count, 0 denotes nil
	vtx         []VertexGroup // ordered vertices
	traces      []int64       // traces storage
	openSlots   []edgeSlot    // unassignerd edge slots within vtx
}

// Recycles this State instance into a pool for reuse.
// Caller asserts that no more references to this instance will persist.
func (X *state) Reclaim() {
	if X != nil {
		X.vtx = X.vtx[:0]
		X.traces = X.traces[:0]
		X.openSlots = X.openSlots[:0]
		graphPool.Put(X)
	}
}

func (X *state) Canonize(normalize bool) error {
	return nil
}

func (X *state) VertexGroups() []VertexGroup {
	return X.vtx
}

func (X *state) WriteCSV(out io.Writer, opts go2x3.PrintOpts) error {
	fmt.Fprintf(out, "p=%d,v=%d,", X.ParticleCount(), X.VertexCount())
	{
		var buf [128]byte
		exprStr, err := X.MarshalOut(buf[:0], go2x3.AsAscii)
		if err != nil {
			return err
		}
		exprStr = append(exprStr, ',')
		out.Write(exprStr)
	}

	if opts.NumTraces != 0 {
		X.WriteTracesAsCSV(out, opts.NumTraces)
	}
	return nil
}

func (X *state) GraphInfo() go2x3.GraphInfo {
	return go2x3.GraphInfo{
		NumParticles: byte(X.ParticleCount()),
		NumVertex:    byte(X.VertexCount()),
	}
}

// Returns the number of particles (partitions) in this graph
func (X *state) ParticleCount() int64 {
	return 1 // TODO
}

func (X *state) PermuteVtxSigns(dst *go2x3.GraphStream) {
	panic("legacy: will not implement")
}

// PermuteEdgeSigns emits a Graph for every possible edge sign permutation of the given
//
// The callback handler should not make any changes to Xperm (with the exception of calling Traces())
func (X *state) PermuteEdgeSigns(dst *go2x3.GraphStream) {

	dst.Outlet <- X.MakeCopy() // TODO

	/*
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
	*/
}

func (X *state) AssemblyMetric() int {
	return X.VertexCount()
}

func (X *state) VertexCount() int {
	if X.vertexCount > 0 {
		return X.vertexCount
	}
	v_produced := int64(0)
	v_consumed := int64(0)
	for _, vi := range X.vtx {
		for _, vi_edge := range vi.Edges {
			switch vi_edge.Direction {
			case EdgeFlow_Intake:
				v_consumed += vi.VertexCount
			case EdgeFlow_Lateral:
				// no change in vertex count
			case EdgeFlow_Outward:
				v_produced += vi.VertexCount
			default:
				panic("illegal edge direction")
			}
		}
	}
	if v_produced != v_consumed {
		panic("vertex count mismatch")
	}
	X.vertexCount = int(X.vtx[0].VertexCount + v_produced)
	return X.vertexCount
}

func (X *state) Traces(wantCycles int) go2x3.Traces {
	Nv := int(X.VertexCount())
	Nt := int(wantCycles)
	if Nt == 0 {
		Nt = Nv
	} else if Nt < 0 {
		panic("negative Traces requested")
	}

	TX := X.traces[:0]

	C1 := make([]Weight, 1+Nv) // over-allocate for 1-based indexing

	// initial state aka identity state
	for i := 0; i <= Nv; i++ {
		vi := &X.vtx[i]
		vi.Ci[i] = Weight{
			Positive: 1,
			Negative: 0,
		}
	}

	// iterate requested number of cycles
	for ti := range Nt {

		// sum inward flow for each vertex for current cycle ci
		totalCycles := Weight{0, 0}
		for i := range Nv {
			vi := &X.vtx[i]

			for j := range Nv {
				vj := &X.vtx[j]
				C1_j := Weight{0, 0}

				switch ti % 2 {
				case 0:
					X.SumEdgeFlows(vi, vj, &C1_j, EdgeFlow_Lateral)
				case 1:
					X.SumEdgeFlows(vi, vj, &C1_j, EdgeFlow_Intake)
				}

				C1[j] = C1_j
			}

			copy(vi.Ci, C1)            // update cycle state
			vi_cycles := C1[i]         // cycle count of vertex vi of length ti
			totalCycles.Add(vi_cycles) // accumulate cycle counts
			vi.Cycles[ti] = vi_cycles
		}

		// TODO: normalize?
		netCycles := totalCycles.Positive - totalCycles.Negative
		TX[ti] = netCycles
		X.traces = append(X.traces, netCycles)

	}

	return TX
}

func (X *state) SumEdgeFlows(vi, vj *VertexGroup, C1_j *Weight, wantDir EdgeFlow) {
	for _, vj_e := range vj.Edges {
		if vj_e.Direction != wantDir {
			continue
		}
		fromIndex := int32(vj.VertexID) - vj_e.FromVertex - 1
		if fromIndex < 0 {
			continue
		}
		// load from state snapshot
		cycles := vi.Ci[fromIndex]

		// sum total flow, scaling by corresponding edge weight
		C1_j.Positive += cycles.Positive * vj_e.Weight.Positive
		C1_j.Negative += cycles.Negative * vj_e.Weight.Negative
	}
}

func (X *state) MakeCopy() go2x3.State {
	return NewState(X)
}

// func (X *state) WriteAsGraphExprStr(out io.Writer) {
// 	for _, vi := range X.Vtx {
// 		fmt.Fprintf(out, "%d:", vi.ID)
// 		for _, ej := range vi.Edges {
// 			if ej.To == 0 {
// 				continue
// 			}
// 			fmt.Fprintf(out, "%d", ej.To)
// 			if ej.Sign == Sign_Invert {
// 				out.Write([]byte{'-'})
// 			} else {
// 				out.Write([]byte{'+'})
// 			}
// 			out.Write([]byte{' '})
// 		}
// 		out.Write([]byte{'\n'})
// 	}
// }

func (X *state) WriteTracesAsCSV(out io.Writer, numTraces int) {
	TX := X.Traces(numTraces)

	var buf [24]byte

	for _, TXi := range TX {
		out.Write(PrintInt(buf[:], TXi))
		out.Write([]byte{','})
	}
}

func (v *VertexGroup) Normalize() {
	if len(v.Edges) == 0 {
		return
	}

	edges := v.Edges
	sort.Slice(edges, func(i, j int) bool {
		flowDir := edges[i].Direction - edges[j].Direction
		if flowDir != 0 {
			return flowDir < 0
		}

		cycles := edges[i].FromVertex - edges[j].FromVertex
		return cycles < 0 // sort by cycle meaning sort by locality
	})

	// when sorted by FromID, a single pass will consolidate like terms
	Ne := len(edges)
	Li := 0
	Li_edge := &edges[Li]
	for Ri := 1; Ri < Ne; Ri++ {
		Ri_edge := &edges[Ri]

		if Li_edge.FromVertex == Ri_edge.FromVertex {
			if Li_edge.Direction != Ri_edge.Direction {
				panic("unexpected edge mismatch")
			}
			Li_edge.Weight.Positive += Ri_edge.Weight.Positive
			Li_edge.Weight.Negative += Ri_edge.Weight.Negative
			Ri_edge.Weight.Positive = 0
			Ri_edge.Weight.Negative = 0
		}

		// drop zero weight ops
		for Ri_edge.IsNoOp() {
			continue
		}

		Li++
		if Li != Ri {
			edges[Li] = *Ri_edge
		}
	}
	v.Edges = edges[:Li]

}

// negative value implies a "future" vertex and so counts as exactly one vertex
// negative values map to +3 instructions, positive values deplete (and return as negaitve)

// func (v *VertexGroup) VertexCountDelta() int64 {
// 	return -v.From * 3
// }

// func (v *VertexGroup) VertexWeight() int64 {
// 	return v.Weight
// }

// func (v *VertexGroup) EmitVariants(chan *VertexGroup) int64 {

// 	// emit all varients of
// }

// 	func (v *VertexGroup) VertexCount() int {
// 		count := 0
// 		if v.SproutNewVertex() {
// 			count += 1 // count spawned vertex as one
// 		}
// 		for _, term := range v.Terms {
// 			if term.From < 0 { // negative value implies a "future" vertex and so counts as exactly one vertex
// 				count++

// 			}
// 			count += term.VertexCountDelta()
// 		}
// }

// func (v *VertexGroup) Traces(numTraces int) go2x3.Traces {

// }

// func (v *VertexGroup) Traces() (odd, even int64) {
// 	for _, term := range v.Terms {
// 		if term.From < 0 {
// 			continue
// 		}
// 		if term.From%2 == 0 {
// 			even += term.Weight
// 		} else {
// 			odd += term.Weight
// 		}
// 	}
// 	return
// }

func (X *state) MarshalOut(out []byte, opts go2x3.MarshalOpts) ([]byte, error) {
	return X.vtx[0].MarshalOut(out, 0, opts)
}

// func (X *state) marshalOut(out []byte, depth int, opts go2x3.MarshalOpts) ([]byte, error) {
// 	for _, vg := range X.groups {
// 		out = vg.MarshalOut(out, opts)
// 	}

// 	return X.root.MarshalOut(out, opts)
// }

func (v *VertexGroup) ParticleCount() int64 {
	return 1 // TODO
}

func (v *VertexGroup) MarshalOut(out []byte, depth int, opts go2x3.MarshalOpts) ([]byte, error) {
	if v.VertexCount == 0 {
		return out, nil
	}
	if v.VertexCount != 1 {
		fmt.Appendf(out, "%d * ", v.VertexCount) // TODO count @ deltaVertex ?
	}

	// sub-terms (edges)
	if len(v.Edges) != 0 {
		out = append(out, '(')
		for _, ve := range v.Edges {
			switch ve.Direction {
			case EdgeFlow_Intake:
				out = append(out, '<') // input edge (from previous cycle)
			case EdgeFlow_Outward:
				out = append(out, '^') // sprout new vertex
			case EdgeFlow_Lateral:
				out = append(out, '-') // same-cycle edge
			}
			fmt.Appendf(out, " (%2d-%2d)", ve.Weight.Positive, ve.Weight.Negative)
		}
		out = append(out, ')')
	}

	/*
		out = append(out, '(')

		vtx := &X.Vtx[vtxID-1]
		for i, ei := range vtx.EdgesOut {
			if ei.Path < 0 { // omit backward edges
				continue
			}

			asPos := byte('o')
			asNeg := byte('@')
			isAddEdge := false
			if ei.To != 0 {
				for j := 0; j < i; j++ {
					ej := vtx.Edges[j]
					if ej.Path > 0 && ej.To == ei.To {
						isAddEdge = true
						asPos = '+'
						asNeg = '-'
						break
					}
				}
			}

			if ei.To == 0 || isAddEdge {
				glyph := byte('?')
				if ei.Sign > 0 {
					glyph = asPos
				} else if ei.Sign < 0 {
					glyph = asNeg
				}
				out = append(out, glyph)
			} else {
				out = X.marshalAsExpr(out, ei.To, asAscii)
			}
		}
		return append(out, ')')
	*/

	return out, nil
}
