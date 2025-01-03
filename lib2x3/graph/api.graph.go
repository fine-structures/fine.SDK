package graph

const (
	EdgesPerVertex = 3
)

type EnumOpts struct {
	VertexMax int
	Params    string
	//Context go2x3.CatalogContext
}

type Edge struct {
	To   VtxID // 1, 2, 3, .. ; 0 denotes nil
	Sign int8  // edge flow scale
	Path int8  // +1: forward, -1: backward
}

// Vertex is a node of a graph, with a fixed number of edges per vertex
type Vertex struct {
	ID    VtxID // 1, 2, 3, ..
	Edges []Edge
}

type VertexGroup struct {
	CycleIndex  int64      // 0, 1, 2, .. -- cycle number when this vertex "cycle" group is traversed
	Occurrences int64      // number of times to repeat this group
	GroupRadius int64      // aka cycle index aka edge distance from the root vertex.
	Edges       []EdgePort // edges flowing into and out of this group
	OpenSlots   []int      // indicies into []EdgesOut of open slots in the group
}

type EdgePort struct {
	WeightPositive int64 // positive weight of this edge
	WeightNegative int64 // negative weight of this edge

	// FromID names which vertex in the previous group this vertex originates from.
	//
	//  -1: SproutsNewEdge
	//   0: SELF_EDGE aka "open slot"
	FromID  int64 // relative index of the inlet from the previous group
	IndexID int64 // 1, 2, 3, ... {positive integer label, 0 denotes nil}
}

const (
	SproutsNewEdge = int64(-1)
	SelfEdge       = int64(0)
	VertexID_1     = int64(1)
	VertexID_2     = int64(2)
	VertexID_3     = int64(3) // etc.
)

// VtxID is one-based index that identifies a vertex in a given graph (1..VtxMax)
type VtxID byte

// CatalogID is a unique identifier for any valid "2x3" graph.
//
// It is an enumeration sequence index, meaning that any graph is a unique number of steps "away" from any other graph.
//
// What the Wolfram project misses is that as a graph grows, is will pass through states such that a coefficient integer "odd even" pair
// can be factored out, where the pair (1,1) is the identity.  This integer pair can be regarded as the count or amplitude of the odd and even Traces terms.
//
// So if (a,b) is a factor, then (n*a, n*b) is the next factor.  This is the same as the Fibonacci sequence, but with a twist.
type CatalogID [10]byte
