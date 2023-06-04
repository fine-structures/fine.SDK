package graph

import (
	"fmt"
	"io"
	"sort"

	"github.com/2x3systems/go2x3/go2x3"
)

type ComputeVtx struct {
	// Two vtx are connected in the same graph if they have the same GraphID.
	GraphID int64

	// When assigned from a 2x3 graph, each Vtx has 3 edges.
	Edges []*EdgeTraces

	// The product of this times VtxCount is the this groups total contribution to VtxGraph.Traces
	Cycles []int64

	VtxID int
	Ci0   []int64
	Ci1   []int64
}

type VtxGraphVM struct {
	VtxGraph

	traces   []int64
	edgePool []*EdgeTraces
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
// 	X.Status = GraphStatus_Invalid
// 	X.Edges = X.Edges[:0]
// 	X.triVtx = nil
// }

// Adds an edge to the vtx / group ID.
// If the named vtx does not exist, it is implicitly created.
func (X *VtxGraphVM) addEdgeToVtx(dst uint32, e *EdgeTraces) {
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
			v.Edges = make([]*EdgeTraces, 0, 8)
			X.vtx[i] = v
		} else {
			v.Edges = v.Edges[:0]
		}
		v.GraphID = int64(i + 1)
		v.VtxID = i + 1
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
	X.Status = GraphStatus_Invalid
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
	X.Status = GraphStatus_Invalid

	// if cap(X.vtx) < Nv {
	// 	X.vtx = make([]*ComputeVtx, Nv, 8 + edgePool2*cap(X.vtx))
	// } else if len(X.vtx) < dstID {
	// 	X.vtx = X.vtx[:dstID]
	// }

	// for i, v := range X.vtx {
	//     if vi == nil {
	//     	v = &ComputeVtx{
	//     	   Edges: make([]*EdgeTraces, 0, 8),
	// 		}
	//     	X.vtx[i] = v
	//     } else {
	// 		v.Edges = v.Edges[:0]
	// 	}
	//     v.VtxID = uint32(i+1)
	// }

}

func (X *VtxGraphVM) newEdge() *EdgeTraces {
	Ne := len(X.edgePool) + 1

	if cap(X.edgePool) < Ne {
		old := X.edgePool
		X.edgePool = make([]*EdgeTraces, Ne, 16+2*cap(X.edgePool))
		copy(X.edgePool, old)
	} else {
		X.edgePool = X.edgePool[:Ne]
	}

	e := X.edgePool[Ne-1]
	if e == nil {
		e = &EdgeTraces{}
		X.edgePool[Ne-1] = e
	} else {
		*e = EdgeTraces{}
		e.Cycles = e.Cycles[:0]
	}
	return e
}

// Adds an edge using one-based indexing.
func (X *VtxGraphVM) AddEdge(
	numNeg, numPos int32,
	vi, vj uint32,
) error {

	if numNeg == 0 && numPos == 0 {
		return nil
	}

	if vi < 1 || vj < 1 {
		return go2x3.ErrInvalidVtxID
	}

	// add a new flow edge for each "side" of the edge
	adding := 1
	if vi != vj {
		adding++
	}

	// Add edge "halves" (one for each vertex to reflect graph flow)
	for i := 0; i < adding; i++ {
		ei := X.newEdge()
		ei.DstVtxID = vi
		ei.SrcVtxID = vj

		count := int64(numPos) - int64(numNeg)
		ei.OddCount = count
		ei.EvenCount = count

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

	vtx := X.Vtx()

	// Keep doing passes until edges do not change GraphID assignments
	for changed := true; changed; {
		changed = false

		for _, ei := range X.edgePool {
			vi_a := ei.DstVtxID
			vi_b := ei.SrcVtxID
			if vi_a == vi_b {
				continue
			}

			va := vtx[vi_a-1]
			vb := vtx[vi_b-1]

			// Propagate the lowest GraphID to the other
			if va.GraphID > vb.GraphID {
				va.GraphID = vb.GraphID
				changed = true
			} else if vb.GraphID > va.GraphID {
				vb.GraphID = va.GraphID
				changed = true
			}
		}
	}

	if err == nil {
		if X.Status < GraphStatus_Validated {
			X.Status = GraphStatus_Validated
		}
		return nil
	}

	return err
}

func (X *VtxGraphVM) Canonize() {

	X.Traces(12)

	// Normalize edge signs -- duplicate every edge we find on a vtx
	{
		edges := X.Edges[:0]

		for _, v := range X.Vtx() {
			for _, src_e := range v.Edges {

				// Edge signs have been "baked" into the cycle signature that we are also going to sign-normalize, so also normalize counts.
				// Drop terms with a normalized count of zero.
				if src_e.OddCount == 0 && src_e.EvenCount == 0 {
					continue
				}

				// Spit edges into even and odd
				{
					e := X.newEdge()

					// Edge signs have been "baked" into the cycle signature that we are also going to sign-normalize, so also normalize counts
					e.GraphID = v.GraphID
					e.OddCount = abs(src_e.OddCount)
					e.EvenCount = abs(src_e.EvenCount)
					e.Cycles = append(e.Cycles[:0], src_e.Cycles...)
					edges = append(edges, e)
				}
			}
		}
		X.Edges = edges
	}

	X.normalize()

}

func compareCycles(a, b *EdgeTraces, isEven int) int64 {
	for i := isEven; i < len(a.Cycles); i += 2 {
		d := a.Cycles[i] - b.Cycles[i]
		if d != 0 {
			return d
		}
	}
	return 0
}

func (X *VtxGraphVM) normalize_signs(isEven int) {

	{
		edges := X.Edges

		// Sign and count normalization
		for _, e := range edges {

			// If a coeff is zero, zero out the elements for clarity
			if isEven != 0 && e.EvenCount == 0 || isEven == 0 && e.OddCount == 0 {
				for i := isEven; i < len(e.Cycles); i += 2 {
					e.Cycles[i] = 0
				}
				continue
			}

			sign := 0 // 0 means not yet determined
			for i := isEven; i < len(e.Cycles); i += 2 {
				ci := e.Cycles[i]

				// find first non-zero cycle and if possible factor out sign to get canonic form
				if sign == 0 && ci != 0 {
					if ci < 0 {
						sign = -1
						if isEven == 1 {
							e.EvenCount = -e.EvenCount
						} else {
							e.OddCount = -e.OddCount
						}
					} else {
						sign = 1
					}
				}
				if sign < 0 {
					e.Cycles[i] = -ci
				}
			}

			// Normalize coeff to 0 if all elements are zero
			if sign == 0 {
				if isEven == 1 {
					e.EvenCount = 0
				} else {
					e.OddCount = 0
				}
			}
		}
	}
}


// For each graph. try to consolidate every possible combo of EdgeTraces
func (X *VtxGraphVM) consolidateEdges() {
	edges := X.Edges
	numEdges := len(edges)

	tryingEdges := make([]*EdgeTraces, numEdges)
	for numToSelect := numEdges; numToSelect >= 2; numToSelect-- {
		n := consolidateEdges(edges, tryingEdges[:0], numToSelect)
		if n > 0 {
			if n%(numToSelect-1) != 0 {
				panic("consolidateEdges: unexpected number of edges removed")
			}
			numEdges -= n
			edges = edges[:numEdges]
		}
	}

	X.Edges = edges

	{
		for _, ei := range edges {

			factorLimit := int64(2701) // בראשית ברא אלהים את השמים ואת הארץ
			for _, ci := range ei.Cycles {
				if ci > 0 && ci < factorLimit {
					factorLimit = ci
				}
			}

			for factor := factorLimit; factor >= 2; factor-- {
				for _, ci := range ei.Cycles {
					if ci%factor != 0 {
						goto nextFactor
					}
				}

				// At this point, we have the highest factor of all cycles
				{
					var count [2]int64
					ei.OddCount *= factor
					ei.EvenCount *= factor
					count[0] = ei.OddCount
					count[1] = ei.EvenCount
					for k, ck := range ei.Cycles {
						ei.Cycles[k] = ck / factor
					}
				}

			nextFactor:
			}
		}
	}
}

// Returns number of edges removed from consolidation (or 0 if none were consolidated)
func consolidateEdges(
	remainEdges []*EdgeTraces, // the edges that are available to be consolidated
	tryingEdges []*EdgeTraces, // edges (by index) that have been chosen so far to try to consolidate
	numToSelect int, // number of edges to select from remainEdges
) int {

	remain := len(remainEdges)
	switch {
	case numToSelect == 0:
		return tryConsolidate(tryingEdges) // try to consolidate the edges we've selected
	case numToSelect > remain:
		return 0 // not enough edges remaining to select from
	case numToSelect == remain:
		{
			tryingEdges = append(tryingEdges, remainEdges...)
			return tryConsolidate(tryingEdges)
		}
	}

	edgesRemoved := 0

	for i := 0; i < remain; i++ {

		// Recurse WITH edge i
		// If tryingEdges[:] was consolidated into tryingEdges[0], back out and restart from tryingEdges[0])
		tryEdges := append(tryingEdges, remainEdges[i])
		n := consolidateEdges(remainEdges[i+1:], tryEdges, numToSelect-1)
		if n > 0 {
			edgesRemoved += n

			if len(tryingEdges) > 0 {

				// Move the now zero-edge ei to an indexes that will be dropped (but retained for pooling)
				for j := remain - 1; j > i; j-- {
					ej := remainEdges[j]
					if ej.OddCount != 0 || ej.EvenCount != 0 {
						remainEdges[j], remainEdges[i] = remainEdges[i], remainEdges[j]
						break
					}
				}
				return edgesRemoved
			}

			// check zero edges are now at the end
			for j := remain - n; j < remain; j++ {
				ej := remainEdges[j]
				if ej.OddCount != 0 || ej.EvenCount != 0 {
					panic("tryingEdges[i] should have been consolidated")
				}
			}

			remain -= n
			remainEdges = remainEdges[:remain]

			// restart from edge i since remainEdges[i] changed
			i--
		}
	}

	return edgesRemoved
}

// pre: for each edge OddCount and EvenCount are non-zero
// Returns how many edges were consolidated (now zeroed out) into edge[0]
// Result will always be 0 or len(edges)-1
func tryConsolidate(edges []*EdgeTraces) int {
	var C [16]int64
	Nc := len(edges[0].Cycles)

	var combined [2]int64
	for _, ei := range edges {
		combined[0] += abs(ei.OddCount)
		combined[1] += abs(ei.EvenCount)
	}

	for k := 0; k < Nc; k++ {
		Ck := int64(0)
		for _, ei := range edges {
			var n int64
			if k&1 == 0 {
				n = ei.OddCount
			} else {
				n = ei.EvenCount
			}
			Ck += n * ei.Cycles[k]
		}
		combinedCount := combined[k&1]
		if Ck%combinedCount != 0 {
			return 0
		}
		C[k] = Ck
	}

	// If we made it here, the traces sum is perfectly divisible by the combined count for each even and ofd
	edges[0].OddCount = combined[0]
	edges[0].EvenCount = combined[1]
	for k := 0; k < Nc; k++ {
		edges[0].Cycles[k] = C[k] / combined[k&1]
	}

	// Zero out edges we consolidated into edge[0]
	for i := 1; i < len(edges); i++ {
		edges[i].OddCount = 0
		edges[i].EvenCount = 0
	}
	return len(edges) - 1

}

func (X *VtxGraphVM) normalize() {

	{
		edges := X.Edges

		// Sort edges by graph they appear in
		sort.Slice(edges, func(i, j int) bool {
			ei := edges[i]
			ej := edges[j]

			// Only terms belonging to the same graph can be consolidated
			if d := ei.GraphID - ej.GraphID; d != 0 {
				return d < 0
			}

			return false
		})
	}

	// TODO: only try to consolidate edge runs that are in the same graph (have the same graph ID)
	X.consolidateEdges()

	X.normalize_signs(0)
	X.normalize_signs(1)

	// Sort edges by normalized + consolidated edge cycles
	{
		edges := X.Edges
		sort.Slice(edges, func(i, j int) bool {
			ei := edges[i]
			ej := edges[j]

			// Only terms belonging to the same graph can be consolidated
			if d := ei.GraphID - ej.GraphID; d != 0 {
				return d < 0
			}

			if d := compareCycles(ei, ej, 0); d != 0 {
				return d < 0
			}
			if d := compareCycles(ei, ej, 1); d != 0 {
				return d < 0
			}
			return false
		})
	}

	/*
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
	   				eL.OddCount += eR.OddCount
	   				eL.EvenCount += eR.EvenCount
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

	   // Sort edges by graph they appear in

	   	sort.Slice(edges, func(i, j int) bool {
	   		ei := edges[i]
	   		ej := edges[j]

	   		// Only terms belonging to the same graph can be consolidated
	   		if d := ei.GraphID - ej.GraphID; d != 0 {
	   			return d < 0
	   		}

	   		// Then sort by cycle signature for even or odd cycles
	   		return cyclesCompare(ei, ej, isEven) < 0
	   	})
	*/
}

/*

type FactorCatalog interface {
	generics.RefCloser

	// Tries to add the given graph encoding to this catalog.
	// Assumes TX is being added in ascending order of NumVertices() since prime detection requires all primes of lesser vertex count to have already been added.
	TryAddTraces(TX TracesProvider) (TracesID, bool)

	// NumTraces returns the number of Traces in this catalog for a given vertex count.
	// An out of bounds vertex count returns 0.
	NumTraces(forVtxCount byte) int64

	// Emits all factorizations of the given Traces using a dynamic programming algorithm to traverse all possible TX partitions.
	cat(TX TracesProvider) <-chan FactorSet
}

*/

func assert(cond bool, desc string) {
	if !cond {
		panic(desc)
	}
}

func (X *VtxGraphVM) Traces(numTraces int) go2x3.Traces {
	if X.Status < GraphStatus_Validated {
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
					netCount := e.EvenCount
					if odd {
						netCount = e.OddCount
					}
					Ci1[j] += netCount * Ci_src

					// Tally cycle returning to the home vtx on this vertex
					if int(vi.VtxID-1) == j {
						if netCount < 0 {
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
		line = append(line, "                   xC1  xC2        "...)

		for ti := range TX {
			ci := ti + 1
			if ci < 10 {
				line = append(line, ' ')
			}
			line = fmt.Appendf(line, "C%d      ", ti+1)
		}

		line = append(line, "\n............................ "...)

		// append traces
		for _, Ti := range TX {
			line = AppendInt(line, Ti, prOpts)
		}

		line = append(line, '\n')
		out.Write(line)
	}

	/*
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

		out.Write([]byte(" -------------------------   \n"))
	*/

	{
		for _, ei := range X.Edges {
			line := ei.AppendDesc(buf[:0])
			line = append(line, "  "...)

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
	//   - use GraphStatus to prevent redundant work
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

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
