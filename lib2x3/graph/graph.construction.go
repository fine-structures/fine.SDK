package graph

import (
	fmt "fmt"
	io "io"
	"sort"
	"sync"

	"github.com/fine-structures/fine.SDK/go2x3"
)

func NewState(src *construction) *construction {
	X := graphPool.Get().(*construction)
	X.traces = X.traces[:0]
	N := 0
	if src != nil {
		X.ForkID = src.ForkID
		X.ParentID = src.ForkID
		X.Cycles = src.Cycles
		X.traces = append(X.traces[:0], src.traces...)
		N = len(src.groups)
	} else {
		X.ForkID = 0
		X.ParentID = 0
		X.Cycles = 0
		X.traces = X.traces[:0]
	}

	if cap(X.groups) >= N {
		X.groups = X.groups[:N]
	} else {
		for len(X.groups) < N {
			X.groups = append(X.groups, VertexGroup{})
		}
	}

	for gi := range X.groups {
		X.groups[gi].CopyFrom(&src.groups[gi])
	}
	return X
}

func (vg *VertexGroup) CopyFrom(vg_in *VertexGroup) {
	vg.Clear()
	if vg_in == nil {
		return
	}

	Ne := len(vg_in.Edges)
	edges := vg.Edges
	if cap(edges) < Ne {
		edges = make([]EdgePort, Ne, (Ne+7)&^7)
	} else {
		edges = append(vg.Edges[:0], vg.Edges...)
	}

	*vg = VertexGroup{
		CycleRadius: vg_in.CycleRadius,
		VertexID:    vg_in.VertexID,
		Occurrences: vg_in.Occurrences,
		Edges:       append(edges, vg_in.Edges...),
		OpenSlots:   append(vg.OpenSlots[:0], vg_in.OpenSlots...),
	}
}

var graphPool = sync.Pool{
	New: func() any {
		return &construction{
			groups: make([]VertexGroup, 0, 24),
			traces: make([]int64, 0, 24),
		}
	},
}

type construction struct {
	ParentID int64         // instance ID
}

func (X *construction) AssemblyMetric() int64 {
	return X.Cycles
}

func (X *construction) Canonize(normalize bool) error {
	return nil
}

func (X *construction) VertexGroups() []VertexGroup {
	return X.groups
}

func (X *construction) WriteCSV(out io.Writer, opts go2x3.PrintOpts) error {
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

func (X *construction) GraphInfo() go2x3.GraphInfo {
	return go2x3.GraphInfo{
		NumParticles: byte(X.ParticleCount()),
		NumVertex:    byte(X.VertexCount()),
	}
}

func (X *construction) VertexCount() int {

	v_produced := int64(0)
	v_consumed := int64(0)
	for _, v_group := range X.groups {
		for _, v_edge := range v_group.Edges {
			posWeight := v_group.Occurrences * (v_edge.WeightPositive + v_edge.WeightNegative)
			switch {

			case v_edge.FromID > 0: // INWARD_EDGE
				vtx_consumed += netWeight
			case v_edge.FromID == 0: // SPROUT_EDGE
				vtx_produced += netWeight
			case v_edge.FromID < 0: // SPROUT_EDGE
				vtx_produced += netWeight
			}
		}
	}
	if v_produced != v_consumed {
		panic("vertex count mismatch")
	}
	return int(v_produced)
}

// Returns the number of particles (partitions) in this graph
func (X *construction) ParticleCount() int64 {
	return 1 // TODO
}

func (X *construction) PermuteVtxSigns(dst *go2x3.GraphStream) {
	panic("legacy: will not implement")
}

// PermuteEdgeSigns emits a Graph for every possible edge sign permutation of the given
//
// The callback handler should not make any changes to Xperm (with the exception of calling Traces())
func (X *construction) PermuteEdgeSigns(dst *go2x3.GraphStream) {

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

const (
	NaturalLength = 0
)

func (X *construction) Traces(wantCycles int) go2x3.Traces {
	Nv := int64(X.VertexCount())
	Nc := int64(wantCycles)
	if Nc == 0 {
		Nc = Nv
	} else if Nc < 0 {
		panic("negative Traces requested")
	}

	if cap(X.traces) < int(Nc) {
		X.traces = make([]int64, (Nc+7)&^7)
	}
	TX := X.traces[:Nc]

	// two state vectors we alternate
	NvNv := Nv * Nv
	C01 := make([]int64, 2*NvNv)
	C0 := C01[:NvNv]
	C1 := C01[NvNv:]

	// init identity state
	for vj := range Nv {
		diagonal := C0[vj*Nv+vj]
		C0[diagonal] = 1 // or any k[vj]
	}

	// iterate requested number of cycles
	for ci := range Nc {

		// sum inward flow for each vertex for current cycle ci
		T_ci := int64(0)
		for vj := range Nv {
			flowSum := int64(0)
			vj_C0 := C0[Nv*ci : Nv*(ci+1)]
			vj_C1 := C1[Nv*ci : Nv*(ci+1)]

			// sum inward flow over all edges
			for _, vj_e := range X.groups[vj].Edges {

				if vj_e.FromID > 0 {
					continue // what sprouts now will count next cycle
				}

				vj_src := vj + vj_e.FromID
				if vj_src < 0 || vj_src >= Nv {
					panic("illegal from vertex")
				}

				flowIn := vj_C0[vj_src] // load flow from previous state
				flowPos := vj_e.WeightPositive * flowIn
				florNeg := vj_e.WeightNegative * flowIn
				flowSum += flowPos - florNeg
			}

			vj_C1[vj] = flowSum // write into next state
			T_ci += flowSum     // accumulate cycle counts of length i
		}

		TX[ci] = T_ci

		// next state becomes current state
		C0, C1 = C1, C0
	}

	return TX
}

func (X *construction) MakeCopy() go2x3.State {
	return NewState(X)
}

// func (X *construction) WriteAsGraphExprStr(out io.Writer) {
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

func (X *construction) WriteTracesAsCSV(out io.Writer, numTraces int) {
	TX := X.Traces(numTraces)

	var buf [24]byte

	for _, TXi := range TX {
		out.Write(PrintInt(buf[:], TXi))
		out.Write([]byte{','})
	}
}

// Recycles this State instance into a pool for reuse.
// Caller asserts that no more references to this instance will persist.
func (X *construction) Reclaim() {
	if X != nil {
		graphPool.Put(X)
	}
}

func (v *VertexGroup) Normalize() {
	if len(v.Edges) == 0 {
		return
	}

	// sort by FromID
	edges := v.Edges
	sort.Slice(edges, func(i, j int) bool {
		d := edges[i].FromID - edges[j].FromID
		return d > 0
	})

	// when sorted by FromID, a single pass will consolidate like terms
	L := 0
	Rn := len(edges) // TODO check me
	for R := 1; R < Rn; R++ {
		if edges[L].FromID == edges[R].FromID {
			edges[L].WeightPositive += edges[R].WeightPositive
			edges[L].WeightNegative += edges[R].WeightNegative
			edges[R].WeightPositive = 0
			edges[R].WeightNegative = 0
		}

		// consolidate zero weight nodes
		if edges[R].WeightPositive != 0 && edges[R].WeightNegative != 0 {
			L++
			if L != R {
				edges[L] = edges[R]
			}
		}
	}
	v.Edges = edges[:L]

	//     else {
	//         j++
	//         edges[j] = edges[i]
	//     }
	// }

	// for i, ei := range edges {
	// 	if ei.From < 0 {
	// 		continue // edges that sprout forward don't stack; they spawn a new vertex "ring"
	// 	}
	// 	for j := i + 1; j < N; j++ {
	// 		if ei.From == edges[j].From {
	// 			ei.Instances += edges[j].Instances
	// 			N--
	// 			edges[j] = edges[N]
	// 			j--
	// 		}
	// 	}
	// }
	// edges = edges[:N]

	// sort.Slice(edges, func(i, j int) bool {
	// 	d := edges[i].From - edges[j].From
	// 	if d != 0 {
	// 		return d > 0 // largest first
	// 	}
	// 	d = edges[i].Instances - edges[j].Instances
	// 	return d > 0
	// })
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

func (v *VertexGroup) Clear() {
	*v = VertexGroup{
		Edges:     v.Edges[:0],
		OpenSlots: v.OpenSlots[:0],
	}
}

func (X *construction) MarshalOut(out []byte, opts go2x3.MarshalOpts) ([]byte, error) {
	return X.groups[0].MarshalOut(out, 0, opts)
}

// func (X *construction) marshalOut(out []byte, depth int, opts go2x3.MarshalOpts) ([]byte, error) {
// 	for _, vg := range X.groups {
// 		out = vg.MarshalOut(out, opts)
// 	}

// 	return X.root.MarshalOut(out, opts)
// }

func (vg *VertexGroup) ParticleCount() int64 {
	return 1 // TODO
}

func (vg *VertexGroup) MarshalOut(out []byte, depth int, opts go2x3.MarshalOpts) ([]byte, error) {
	if vg.Occurrences == 0 {
		return out, nil
	}
	if vg.Occurrences != 1 {
		fmt.Appendf(out, "%d * ", vg.Occurrences) // TODO count @ deltaVertex ?
	}

	// sub-terms (edges)
	if len(vg.Edges) != 0 {
		out = append(out, '(')
		for _, ve := range vg.Edges {
			if ve.FromID < 0 {
				out = append(out, "^"...) // sprout new vertex
			} else if ve.FromID == 0 {
				out = append(out, "*"...) // self edge
			} else {
				fmt.Appendf(out, "@%-2d", ve.FromID)
			}
			fmt.Appendf(out, " (%2d-%2d)", ve.WeightPositive, ve.WeightNegative)
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
