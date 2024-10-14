package graph

import (
	"github.com/fine-structures/fine-sdk-go/lib2x3/mesh"
)

const (
	EdgesPerVertex = 3
)

type EnumOpts struct {
	VertexMax int
	Params    string
	//Context go2x3.CatalogContext
}

type Edge struct {
	To     VtxID       // 1, 2, 3, .. ; 0 denotes nil
	SlotID int8        // cardinal direction; 1, 2, or 3; 0 denotes nil / unassigned
	Flow   int8        // cardinal amplitude
	Op     mesh.OpCode // edge type / mode
}

// Vertex is a node of a graph, with a fixed number of edges per vertex
type Vertex struct {
	ID    VtxID // 1, 2, 3, ..
	Edges [EdgesPerVertex]Edge
}

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
