package graph

import (
	"fmt"
	"io"
	"math"
	"sort"

	"github.com/fine-structures/fine-sdk-go/go2x3"
)

const (
	VertexBits    = 5
	VertexBitMask = (1 << VertexBits) - 1
	MaxVertexID   = VertexBitMask

	IsPrimeBit  = 1 << 6
	ReservedBit = 1 << 7
)

// Returns the
func (Xi *GraphTerm) CatalogID() CatalogID {
	Nv := Xi.VertexCount
	if Nv < 1 || Nv > MaxVertexID ||
		Xi.IsPrime == Bool_Unknown ||
		Xi.IsCanonic == Bool_Unknown ||
		Xi.StructureID < 1 {

		//return GraphID{}
		panic("invalid GraphTerm")
	}

	cid := CatalogID{
		byte(Nv),
		byte(Xi.IsPrime),
		byte(Xi.StructureID >> 24),
		byte(Xi.StructureID >> 16),
		byte(Xi.StructureID >> 8),
		byte(Xi.StructureID),
		byte(Xi.VariantID >> 24),
		byte(Xi.VariantID >> 16),
		byte(Xi.VariantID >> 8),
		byte(Xi.VariantID),
	}
	return cid
}

func (Xi *GraphTerm) PrimeID() (pid PrimeID, ok bool) {
	if Xi.IsPrime != Bool_Yes {
		return PrimeID(0), false
	}
	return FormPrimeID(Xi.VertexCount, Xi.StructureID, Xi.VariantID), true
}

// A unique identifier for a canonical prime graph.
//
//	     Nv     StructureID     VariantID
//	    -----------------------------------
//	0x   FF      FF FF FF       FF FF FF FF
type PrimeID uint64

func FormPrimeID(Nv uint32, structureID, variantID uint64) PrimeID {
	if Nv < 1 || Nv > MaxVertexID ||
		structureID < 1 || structureID > 0x00ffffff || variantID > 0xffffffff {
		panic("invalid PrimeID")
	}
	pid := PrimeID(Nv)<<56 | PrimeID(structureID)<<32 | PrimeID(variantID)
	return pid
}

func (pid PrimeID) CatalogID() CatalogID {
	cid := CatalogID{
		byte(pid.VertexCount()),
		byte(Bool_Yes),
		byte(0),
		byte(pid >> 48),
		byte(pid >> 40),
		byte(pid >> 32),
		byte(pid >> 24),
		byte(pid >> 16),
		byte(pid >> 8),
		byte(pid),
	}
	return cid
}

// Extracts the number of vertices embedded in a PrimeID
func (pid PrimeID) VertexCount() uint32 {
	return uint32(pid >> 56)
}

// Extracts the ID of the structure variant tuple ID that uniquely corresponds to a canonical prime graph.
func (pid PrimeID) PrimeCatalogID() uint64 {
	return uint64(pid) & 0x00ffffffffffffff
}

type ComputeVtx struct {
	VtxGroup

	// Initially assigned label: 1, 2, 3, ..  (one-based index)
	VtxID uint32

	Ci0 []int64 // trace in place
	Ci1 []int64 // trace in place
}

type VtxGraphVM struct {
	Status GraphStatus

	edgeCount int        // allocated edges: edgePool[:edgeCount]
	edgePool  []*VtxEdge // used and non-used edges
	traces    []int64
	calcBuf   []int64
	vtx       []*ComputeVtx // Vtx by VtxID (zero-based indexing)
	vtxMap    []uint32      // original VtxID to consolidated VtxID (zero-based indexing)
}

const maxNv = 18

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

/*
// Combines edges with the same src vtx and normalizes order
func (v *VtxGroup) normalizeEdges() {
	edges := v.Edges
	Ne := len(edges)

	// Combine same-src edges
	for i := 0; i < Ne; i++ {
		src_i := edges[i].SrcVtxID
		for j := i + 1; j < Ne; j++ {
			if edges[j].SrcVtxID == src_i {
				edges[i].Count += edges[j].Count
				edges[j] = edges[Ne-1]
				Ne--
				j--
			}
		}
	}

	// Drop zero count edges
	for i := 0; i < Ne; i++ {
		if edges[i].Count == 0 {
			edges[i] = edges[Ne-1]
			Ne--
			i--
		}
	}
	edges = edges[:Ne]

	sort.Slice(edges, func(i, j int) bool {
		d := int(edges[i].SrcVtxID) - int(edges[j].SrcVtxID)
		return d < 0
	})
	v.Edges = edges
}
*/

// Adds an edge to the vtx / group ID.
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
		Nv++
		v.Count = 1
		v.GroupID = 0
		v.OddSign = OddSign_Natural
		v.GraphID = uint32(Nv)
		v.VtxID = uint32(Nv)
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

	X.vtx = X.vtx[:0]
	X.edgeCount = 0
	X.traces = nil
	X.Status = GraphStatus_Invalid
}

func (X *VtxGraphVM) newEdge() *VtxEdge {
	Ne := X.edgeCount

	if cap(X.edgePool) <= Ne {
		old := X.edgePool
		X.edgePool = make([]*VtxEdge, 16+2*cap(X.edgePool))
		copy(X.edgePool, old)
	}

	e := X.edgePool[Ne]
	if e == nil {
		e = &VtxEdge{}
		X.edgePool[Ne] = e
	} else {
		*e = VtxEdge{}
	}
	X.edgeCount = Ne + 1
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
		ei.Count = count

		X.addEdgeToVtx(vi, ei)

		// Add the other flow edge to the other vertex
		if adding == 2 {
			vi, vj = vj, vi
		}
	}

	return nil
}

func (X *VtxGraphVM) Validate() error {
	vtx := X.Vtx()

	// Keep doing passes until edge propagation doesn't change GraphID assignments
	for changed := true; changed; {
		changed = false

		for _, vi := range vtx {
			for _, ej := range vi.Edges {
				vj_ID := ej.SrcVtxID
				if vj_ID == vi.VtxID {
					continue
				}

				// Propagate the lowest GraphID to the other
				vj := vtx[vj_ID-1]
				if vi.GraphID > vj.GraphID {
					vi.GraphID = vj.GraphID
					changed = true
				} else if vj.GraphID > vi.GraphID {
					vj.GraphID = vi.GraphID
					changed = true
				}
			}
		}
	}

	// re-index GraphID to be sequential
	{
		remap := make([]byte, len(vtx)+1)
		N := byte(0)
		for _, v := range vtx {
			if remap[v.GraphID] == 0 {
				N++
				remap[v.GraphID] = N
			}
		}
		for _, v := range vtx {
			v.GraphID = uint32(remap[v.GraphID])
		}
	}
	if X.Status < GraphStatus_Validated {
		X.Status = GraphStatus_Validated
	}
	return nil
}

func (X *VtxGraphVM) Canonize() {
	X.Traces(24)
	X.consolidateVtx()
	X.normalize()
}

func compareCycles(a, b *ComputeVtx) int64 {
	for i, ai := range a.Cycles {
		d := ai - b.Cycles[i]
		if d != 0 {
			return d
		}
	}
	return 0
}

func (X *VtxGraphVM) normalize() {
	vtx := X.Vtx()

	// Sign and count normalization
	for _, v := range vtx {

		// If a coeff is zero, zero out the elements for clarity
		if v.Count == 0 {
			for i := range v.Cycles {
				v.Cycles[i] = 0
			}
			continue
		}

		// Normalize odd sign
		sign := OddSign_Zero
		for i := 0; i < len(v.Cycles); i += 2 {
			ci := v.Cycles[i]

			// find first non-zero cycle and if possible factor out sign to get canonic form
			if sign == OddSign_Zero && ci != 0 {
				if ci < 0 {
					sign = OddSign_Invert
				} else {
					sign = OddSign_Natural
					break
				}
			}
			if sign == OddSign_Invert {
				v.Cycles[i] = -ci
			}
		}
		v.OddSign = sign
	}

	// Sort vtx by normalized + consolidated cycles
	sort.Slice(vtx, func(i, j int) bool {
		vi := vtx[i]
		vj := vtx[j]

		// Only terms belonging to the same graph can be consolidated
		if d := int(vi.GraphID) - int(vj.GraphID); d != 0 {
			return d < 0
		}
		if d := compareCycles(vi, vj); d != 0 {
			return d < 0
		}
		return false
	})

	// Conically assign GroupID
	{
		groupCount := uint32(0)
		curGraphID := uint32(0)
		for _, vi := range vtx {
			if vi.GraphID != curGraphID {
				groupCount = 0
				curGraphID = vi.GraphID
			}
			groupCount++
			vi.GroupID = groupCount
		}
	}

	// Reassign VtxID to be final group ID
	{
		vtxToGrpID := make([]uint32, len(X.vtxMap))
		for wasVtxID, nowVtxID := range X.vtxMap {
			for _, vi := range vtx {
				if vi.VtxID == nowVtxID {
					vtxToGrpID[wasVtxID] = vi.GroupID
					break
				}
			}
		}

		// Reassign all VtxIDs to be the final group ID
		for _, vi := range vtx {
			vi.VtxID = vtxToGrpID[vi.VtxID-1]
			for _, ej := range vi.Edges {
				ej.DstVtxID = vtxToGrpID[ej.DstVtxID-1]
				ej.SrcVtxID = vtxToGrpID[ej.SrcVtxID-1]
			}
		}
	}

	// for _, vi := range vtx {
	// 	vi.normalizeEdges()
	// }
}

// For each graph. try to consolidate every possible combo of VtxGroup
func (X *VtxGraphVM) consolidateVtx() {
	vtx := X.vtx
	Nv := len(vtx)

	tryingVtx := make([]*ComputeVtx, Nv)
	for numToSelect := Nv; numToSelect >= 2; numToSelect-- {
		n := X.consolidateVtxRecurse(vtx, tryingVtx[:0], numToSelect)
		if n > 0 {
			if n%(numToSelect-1) != 0 {
				panic("consolidateVtx: unexpected number of vtx removed")
			}
			Nv -= n
			vtx = vtx[:Nv]
		}
	}

	X.vtx = vtx
	/*
	   // Factor out greatest common factor from each vtx traces

	   	{
	   		for _, vi := range vtx {

	   			// The smallest non-zero cycle value is the max GCF, so start there
	   			factorLimit := int64(2701 * 1072)
	   			for _, ci := range vi.Cycles {
	   				if ci > 0 && ci < factorLimit {
	   					factorLimit = ci
	   				}
	   			}

	   			for factor := factorLimit; factor >= 2; factor-- {
	   				for _, ci := range vi.Cycles {
	   					if ci%factor != 0 {
	   						goto nextFactor
	   					}
	   				}

	   				// At this point, we have the highest factor of all cycles
	   				{
	   					vi.Count *= factor
	   					for k, ck := range vi.Cycles {
	   						vi.Cycles[k] = ck / factor
	   					}
	   				}

	   			nextFactor:
	   			}
	   		}
	   	}
	*/
}

// Returns number of vtx removed from consolidation (or 0 if none were consolidated)
func (X *VtxGraphVM) consolidateVtxRecurse(
	remainVtx []*ComputeVtx, // the vtx that are available to be consolidated
	tryingVtx []*ComputeVtx, // vtx (by index) that have been chosen so far to try to consolidate
	numToSelect int, // number of vtx to select from remainVtx
) int {

	remain := len(remainVtx)
	vtxRemoved := 0
	baseCase := true

	switch {
	case numToSelect == 0:
		vtxRemoved = tryConsolidate(tryingVtx) // try to consolidate the vtx we've selected
	case numToSelect > remain:
		vtxRemoved = 0 // not enough vtx remaining to select from
	case numToSelect == remain:
		tryingVtx = append(tryingVtx, remainVtx...)
		vtxRemoved = tryConsolidate(tryingVtx)
	default:
		baseCase = false
	}

	if baseCase {
		if vtxRemoved > 0 { // Zero out the consolidated vtx
			v0 := tryingVtx[0]
			newID := v0.VtxID
			for _, vi := range tryingVtx[1:] {
				oldID := vi.VtxID
				X.vtxMap[oldID-1] = newID
				vi.Count = 0

				// Absorb edges from vi into v0
				v0.Edges = append(v0.Edges, vi.Edges...)

				// Update any vtx that pointed to vi to point to v0
				for j, vj := range X.vtxMap {
					if vj == oldID {
						X.vtxMap[j] = newID
					}
				}
			}
		}

		return vtxRemoved
	}

	for i := 0; i < remain; i++ {

		// Cull work: consolidation is only possible for vtx in the same graph
		if len(tryingVtx) > 0 && tryingVtx[0].GraphID != remainVtx[i].GraphID {
			continue
		}
		// Recurse WITH vtx i
		// If tryingVtx[:] was consolidated into tryingVtx[0], back out and restart from tryingVtx[0])
		tryVtx := append(tryingVtx, remainVtx[i])
		n := X.consolidateVtxRecurse(remainVtx[i+1:], tryVtx, numToSelect-1)
		if n > 0 {
			vtxRemoved += n

			if len(tryingVtx) > 0 {

				// Move the now zeroed vtx to indexes to be dropped (but retained for pooling)
				for j := remain - 1; j > i; j-- {
					ej := remainVtx[j]
					if ej.Count != 0 {
						remainVtx[j], remainVtx[i] = remainVtx[i], remainVtx[j]
						break
					}
				}
				return vtxRemoved
			}

			// check zero vtx are now at the end
			for j := remain - n; j < remain; j++ {
				ej := remainVtx[j]
				if ej.Count != 0 {
					panic("tryingVtx[i] should have been consolidated")
				}
			}

			remain -= n
			remainVtx = remainVtx[:remain]

			// restart from vtx i since remainVtx[i] changed
			i--
		}
	}

	return vtxRemoved
}

// pre: for each vtx OddCount and EvenCount are non-zero
// Returns how many vtx were consolidated into vtx[0] -- always either 0 or len(vtx)-1
func tryConsolidate(vtx []*ComputeVtx) int {
	var C [32]int64
	Nc := len(vtx[0].Cycles)

	combined := int64(0)
	graphID := vtx[0].GraphID
	for _, vi := range vtx {
		combined += vi.Count
		if graphID != vi.GraphID {
			return 0
		}
	}

	for k := 0; k < Nc; k++ {
		Ck := int64(0)
		for _, vi := range vtx {
			n := vi.Count
			if k&1 == 0 && vi.OddSign == OddSign_Invert {
				n = -n
			}
			Ck += n * vi.Cycles[k]
		}
		if Ck%combined != 0 {
			return 0 // if cycles sum not divisible by the combined count, we cannot consolidate
		}
		C[k] = Ck
	}

	// At this point, the traces sum is perfectly divisible by the combined count for each even and ofd
	vtx[0].OddSign = OddSign_Natural
	vtx[0].Count = combined
	for k := 0; k < Nc; k++ {
		vtx[0].Cycles[k] = C[k] / combined
	}

	return len(vtx) - 1
}

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
		X.vtxMap[i] = uint32(i + 1)
	}

	// Oh Lord, our Adonai and God, you alone are the Lord. You have made the heavens, the heaven of heavens, with all their host, the earth and all that is on it, the seas and all that is in them; and you preserve all of them; and the host of heaven worships you. You are the Lord, the God, who chose Abram and brought him out of Ur of the Chaldeans and gave him the name Abraham; you found his heart faithful before you, and made with him the covenant to give the land of the Canaanites, the Hittites, the Amorites, the Perizzites, the Jebusites, and the Girgashites—to give it to his offspring. You have kept your promise, for you are righteous. And you saw the affliction of our fathers in Egypt and heard their cry at the Red Sea; and you performed signs and wonders against Pharaoh and all his servants and all the people of his land, for you knew that they acted arrogantly against them. And you made a name for yourself, as it is this day, and you divided the sea before them, so that they went through the midst of the sea on dry land, and you cast their pursuers into the depths, as a stone into mighty waters. Moreover in a pillar of cloud you led them by day, and in a pillar of fire by night, to light for them the way in which they should go. You came down also upon Mount Sinai, and spoke with them from heaven, and gave them right ordinances and true laws, good statutes and commandments; and you made known to them your holy sabbath, and commanded them commandments and statutes, a law for ever. And you gave them bread from heaven for their hunger, and brought forth water for them out of the rock for their thirst, and you told them to go in to possess the land that you had sworn to give them. But they and our fathers acted presumptuously and stiffened their neck, and did not obey your commandments. They refused to obey, neither were mindful of the wonders that you performed among them, but hardened their necks, and in their rebellion appointed a leader to return to their bondage. But you are a God ready to pardon, gracious and merciful, slow to anger, and abounding in steadfast love, and did not forsake them. Even when they had made for themselves a calf of molten metal, and~.
	// Yashua is His name, Emmanuel, God with us!
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
					netCount := e.Count
					Ci1[j] += netCount * Ci_src
				}
			}

			vi_cycles_ci := Ci1[vi.VtxID-1]
			X.traces[ci] += vi_cycles_ci
			vi.Cycles[ci] = vi_cycles_ci
		}
	}
}

var (
	gLineSep = "........."
)

func T_ceiling_240303(Nv int, Ci int, prevTraces int64) float64 {
	scale := float64(1) //float64(Nv)
	if prevTraces != 0 {
		scale *= float64(prevTraces)
		if prevTraces < 0 {
			scale = -scale
		}
	}
	return scale
}

/*
	func T_ceiling_240303(Nv int, Ci int, prevC0, prevC1 int64) (float64, float64) {
		scale := float64(1) // float64(Nv)
		if (prevTraces != 0) {
			scale *= float64(prevTraces)
			if prevTraces < 0 {
				scale = -scale
			}
		}
		return scale
	}
*/
func T_FibLucas(Nv int, Ci int, Fn0, Fn1 int64) float64 {
	Fn2 := Fn0 + Fn1
	Fn2_norm := math.Log(float64(Fn2))
	return Fn2_norm
}

func T_NormEclipse2024(Nv int, Ci int, Fn0, Fn1 int64) float64 {
	Fn2 := Fn0 + Fn1
	Fn2_norm := math.Log(float64(Fn2))
	return Fn2_norm
}

// func T_NormEclipse2024_alt(Nv int, Ci int, Fn0, Fn1 int64) float64 {
// 	C_norm := float64(1)
// 	for i := 0; i < Ci; i++ {
// 		C_norm *= 3
// 	}
// 	f0 :=  float64(Fn0) * C_norm
// 	f1 := -float64(Fn1) * C_norm

// 	sign := float64(1)
// 	if Ci%2 == 1 {
// 		sign = -1
// 	}

// 	Fn2 := Fn0 + Fn1
// 	Fn2_norm := math.Log(float64(Fn2))
// 	return Fn2_norm
// }

// func Factorize(factorCounts *[]int64, primeFactors *[]int64, x int64) {

func PrintNormalizedTraces(Nv int, TX go2x3.Traces, out io.Writer) {

	//Xv := X.Vtx()
	//Nc := len(TX)

	Nc := len(TX)
	var buf [512]byte
	f64Buf := make([]float64, 0, 32)
	TX_norm := f64Buf[:Nc]
	// prOpts := PrintIntOpts{
	// 	MinWidth: len(gLineSep),
	// }

	// Write header
	{
		// factorCounts := make([]int64, 0, 32)
		// primeFactors := make([]int64, 0, 32)

		// Print TX entries
		{
			line := buf[:0]
			//Tn0 := int64(0)
			for ci := 0; ci < Nc; ci++ {
				Tn1 := TX[ci]
				//TX_norm[ci] = T_FibLucas(Nv, ci, Tn0, Tn1)
				TX_norm[ci] = float64(Tn1) //float64(Tn0 + Tn1)
				//Tn0 = Tn1
			}

			//Tn1_norm := 1 / TX_norm[0]

			Tn0_norm := float64(+1) / float64(3*Nv)
			Tn1_norm := float64(-1) / float64(3*Nv)

			for ci := 0; ci < Nc; ci++ {
				TX_norm[ci] = Tn0_norm + Tn1_norm
				Tn0_norm = Tn1_norm
				Tn1_norm *= Tn1_norm
			}

			// if TX[0] != 0 {
			// 	Tn0_norm = 1.0 / float64(TX[0])
			// }
			// if TX[1] != 0 {
			// 	Tn1_norm = 1.0 / float64(TX[1])
			// }

			// for isEven := 0; isEven < 2; isEven++ {
			// 	for ci := isEven ; ci < Nc; ci +=2 {

			// 	}
			// }

			// 	C_norm := float64(1)
			// 	for i := 0; i < Ci; i++ {
			// 		C_norm *= 3
			// 	}
			// for ci := Nc - 2; ci > 0; ci-- {
			// 	norm := T_norm
			// 	if TX_norm[ci] != 0 {
			// 		norm /= TX_norm[ci]
			// 	}
			// 	TX_norm[ci] = norm * TX_norm[ci+1]
			// }

			// for i := 0; i < Nc/2; i += 2 {

			// }

			// TODO: explore Codd being a modulus of 3^Ci (aka the particle integer "characteristic")
			//  and Ceven being floating point energy "characteristic")
			for isEven := 0; isEven < 2; isEven++ {
				for ci := isEven; ci < Nc; ci += 2 {
					Ti := TX_norm[ci] //TX[ci]
					//A := Ti % 3
					// if i == isEven {
					// 	if Ti != 0 {
					// 		norm = 1.0 / A
					// 		A = 1.0
					// 	}
					// } else {
					// 	A *= norm
					// }
					line = fmt.Appendf(line, " C%02d: %+9d  %+9f   ", ci, TX[ci], Ti)

					// factorCounts = factorCounts[:0]
					// primeFactors = primeFactors[:0]
					// graph.Factorize(&factorCounts, &primeFactors, Ti)
					// for i, factor := range primeFactors {
					// 	line = fmt.Appendf(line, " %03d %-3d  ", factorCounts[i], factor)
					// }

					line = fmt.Appendf(line, " \n")

					// //line = fmt.Appendf(line, " %+7.4f  ", A)
					// Tn0 = Tn1
					// Tn1 = Ti
				}

				// norm := float64(1)
				// if A != 0 {
				// 	norm = 1.0 / A
				// }
				// Tprev = 0
				// for i := isEven ; i < Nc; i +=2 {
				// 	denom := T_ceiling_240303(Nv, i+1, Tprev)
				// 	Ti := TX[i]
				// 	A = norm * float64(Ti) / denom
				// 	line = fmt.Appendf(line, " %+7.4f  ", A)
				// 	Tprev = Ti
				// }

				line = append(line, '\n')
			}

			// func T_ceiling_240303(Nv int, Ci int, prevC0, prevC1 int64) float64 {
			/*
				prevC0 := int64(0)
				prevC1 := int64(0)
				// for ci := 0; ci < Nc; ci += 2 {
				// 	denom := T_ceiling_240303(Nv, ci, prevC0, prevC1)
				// }
				for i, Ti := range TX {
					if (Tprev != 0) {
						scale *= float64(Tprev)
						if Tprev < 0 {
							scale = -scale
						}
					}
					// denom := T_ceiling_240303(Nv, i+1, Tprev)
					// deflection := float64(Ti) / denom
					// fmt.Printf(" %-7.3f  ", deflection)
					// line = fmt.Appendf(line, " %-7.3f  ", deflection)

					if i%2 == 0 {
						prevC0 = Ti
					} else {
						prevC1 = Ti
						line = append(line, '\n')
					}
				}
			*/
			out.Write(line)
		}

		// line = append(line, "\n----------            "...)
		// for i := 0; i < Nc; i++ {
		// 	line = append(line, gLineSep...)
		// }
		// line = append(line, '\n')

		// out.Write(line)
	}
	/*
		{
			for _, vi := range X.Vtx() {
				//for ni := int64(0); ni < ei.Count; ni++ {
				{
					line := vi.AppendDesc(buf[:0])
					line = append(line, "  "...)
					for i := 0; i < Nc; i++ {
						line = AppendInt(line, vi.Cycles[i], prOpts)
					}
					line = append(line, '\n')
					// for _, ej := range vi.Edges {
					// 	dst := byte('A') - 1 + byte(ej.DstVtxID)
					// 	src := byte('A') - 1 + byte(ej.SrcVtxID)
					// 	line = fmt.Appendf(line, "  %c  <=  %3d x %c \n", dst, ej.Count, src)
					// }
					out.Write(line)
				}
			}
		}
	*/
}

func (X *VtxGraphVM) PrintCycleSpectrum(numTraces int, out io.Writer) {
	TX := X.Traces(numTraces)

	//Xv := X.Vtx()
	Nc := len(TX)

	var buf [128]byte

	prOpts := PrintIntOpts{
		MinWidth: len(gLineSep),
	}

	// Write header
	{
		line := buf[:0]
		line = append(line, "                 ##        "...)

		for ti := range TX {
			ci := ti + 1
			if ci < 10 {
				line = append(line, ' ')
			}
			line = fmt.Appendf(line, "C%d      ", ti+1)
		}

		// append traces
		line = append(line, "\n                     "...)
		for _, Ti := range TX {
			line = AppendInt(line, Ti, prOpts)
		}

		line = append(line, "\n                     "...)
		for i := 0; i < Nc; i++ {
			line = append(line, gLineSep...)
		}
		line = append(line, '\n')

		out.Write(line)
	}
	{
		for _, vi := range X.Vtx() {
			//for ni := int64(0); ni < ei.Count; ni++ {
			{
				line := vi.AppendDesc(buf[:0])
				line = append(line, "  "...)
				for i := 0; i < Nc; i++ {
					line = AppendInt(line, vi.Cycles[i], prOpts)
				}
				line = append(line, '\n')
				// for _, ej := range vi.Edges {
				// 	dst := byte('A') - 1 + byte(ej.DstVtxID)
				// 	src := byte('A') - 1 + byte(ej.SrcVtxID)
				// 	line = fmt.Appendf(line, "  %c  <=  %3d x %c \n", dst, ej.Count, src)
				// }
				out.Write(line)
			}
		}
	}

}

/*
func (X *VtxGraphVM) RegenNormalizedTraces() {

//PrintNormalizedTraces(X.VertexCount(), X.Traces(16), out)

//Xv := X.Vtx()
//Nc := len(TX)

var buf [512]byte

// prOpts := PrintIntOpts{
// 	MinWidth: len(gLineSep),
// }

// Write header
{
	line := buf[:0]
	line = append(line, "                 ##        "...)

	// Print TX entries
	{
		line := buf[:0]
		Nv := len(X.Vtx())

		for isEven := 0; isEven < 2; isEven++ {
			Tprev := int64(0)
			Nc := len(TX)
			//norm := float64(1)
			for i := isEven; i < Nc; i += 2 {
				denom := T_ceiling_240303(Nv, i+1, Tprev)
				Ti := TX[i]
				A := float64(Ti) / denom
				// if i == isEven {
				// 	if Ti != 0 {
				// 		norm = 1.0 / A
				// 		A = 1.0
				// 	}
				// } else {
				// 	A *= norm
				// }

				line = fmt.Appendf(line, " %+7.4f  ", A)
				Tprev = Ti
			}
			line = append(line, '\n')
		}

		// for i, Ti := range TX {

		// 	denom := T_ceiling_240303(Nv, i+1, Tprev)
		// 	deflection := float64(Ti) / denom
		// 	fmt.Printf(" %-7.3f  ", deflection)
		// 	line = fmt.Appendf(line, " %-7.3f  ", deflection)

		// 	if i%2 == 0 {
		// 		Tprev[0] = Ti
		// 	} else {
		// 		Tprev[1] = Ti
		// 		line = append(line, '\n')
		// 	}
		// }
		out.Write(line)
	}

	line = append(line, "\n----------            "...)
	// for i := 0; i < Nc; i++ {
	// 	line = append(line, gLineSep...)
	// }
	// line = append(line, '\n')

	// out.Write(line)
}
*/

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
	}

	if cap(X.vtxMap) < Nv {
		X.vtxMap = make([]uint32, Nv, maxNv)
	} else {
		X.vtxMap = X.vtxMap[:Nv]
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
