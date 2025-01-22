package graph

import "github.com/fine-structures/fine.SDK/go2x3"

const (
	EdgesPerVertex = 3
)

type EdgeOp struct {
	Weight    Weight   // scales flow of this edge each cycle
	Direction EdgeFlow // direction of the flow of this edge, needed??

	// vertex index offset from which vertex to pull state from each cycle.
	//
	//   .. 3, 2, 1, (existing)
	//      0 (self),
	//     -1 (sprout),
	//     -2 (sprout),
	//     -3 (sprout), ..
	FromVertexOffset int32
}

func (edge EdgeOp) IsNoOp() bool {
	return edge.Weight.Positive == 0 && edge.Weight.Negative == 0
}

type EnumOpts struct {
	VertexMax int
	Params    string
	Context   go2x3.CatalogContext
}

type VertexGroup struct {
	VertexID    int32    // 1, 2, 3, ..   positive integer label; 0 == nil
	GroupID     int32    // 1, 2, 3, ..   positive integer label; 0 == nil
	CycleRadius int64    // 0, 1, 2, ..   traces index; corresponds to iteration distance from the root vertex
	VertexCount int64    // number of instances this group
	Edges       []EdgeOp // edges flowing inward from previous cycle group, laterally from this cycle group, and outward =
	Cycles      []Weight // cycles as a function of cycle radius (1, 2, 3, ...)
	Ci          []Weight // current cycle state
}

type EdgeFlow int

const (
	EdgeFlow_Intake  EdgeFlow = -1 // input edge (from previous cycle)
	EdgeFlow_Lateral EdgeFlow = 0  // intra-cycle edge
	EdgeFlow_Outward EdgeFlow = +1 // outward edge (new vertex)
)

type Weight struct {
	Positive int64
	Negative int64
}

func (w Weight) IsZero() bool {
	return w.Positive == 0 && w.Negative == 0
}

func (w *Weight) Add(other Weight) {
	w.Positive += other.Positive
	w.Negative += other.Negative
}

func (w *Weight) AddWeighted(other Weight, weight Weight) {
	w.Positive += other.Positive * weight.Positive
	w.Negative += other.Negative * weight.Negative
}

// CatalogID is a unique identifier for any valid "2x3" graph.
//
// It is an enumeration sequence index, meaning that any graph is a unique number of steps "away" from any other graph.
//
// What the Wolfram project misses is that as a graph grows, is will pass through states such that a coefficient integer "odd even" pair
// can be factored out, where the pair (1,1) is the identity.  This integer pair can be regarded as the count or amplitude of the odd and even Traces terms.
//
// So if (a,b) is a factor, then (n*a, n*b) is the next factor.  This is the same as the Fibonacci sequence, but with a twist.
type CatalogID [10]byte
