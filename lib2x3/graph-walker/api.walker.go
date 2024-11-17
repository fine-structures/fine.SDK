package walker

import (
	"github.com/fine-structures/fine.SDK/go2x3"
	"github.com/fine-structures/fine.SDK/lib2x3/graph"
)

// Primary entry point for "2x3" graph enumeration aka "fine structures"
func EnumPureParticles(opts EnumOpts) (*go2x3.GraphStream, error) {
	return enumPureParticles(opts)
}

// OpCode is a graph-building op code, specifying a way to grow a 2x3 graph.
type OpCode uint8

const (
	OpCode_AddEdge OpCode = 1 // Places an additional edge across the vertices for a given edge
	OpCode_Sprout  OpCode = 2 // Sprouts an new edge (and vertex) from a given vertex slot
	// OpCode_MirrorGraph   OpCode = 3 // Duplicates the graph and adds an edge to each corresponding pair of vertices
	// OpCode_ExpandVertex OpCode = 4 // Replaces a given vertex with a ring of connected vertices, each vertex preserve the originating vertex's endpoints

)

// GrowOp is a graph building step, specifying exactly where and how to form a new edge.
type GrowOp struct {
	OpCode   OpCode      // operation to perform
	Count    int8        // number of times to perform the operation (typically +1 or -1)
	FromVtx  graph.VtxID // 1, 2, 3, .. ; 0 denotes nil
	FromSlot uint8       // 0, 1, 2
}

func (op *GrowOp) FromOrdinal() int {
	return int(op.FromVtx)*graph.EdgesPerVertex + int(op.FromSlot)
}

type EnumOpts struct {
	VertexMin int
	VertexMax int
	Params    string
	//Context go2x3.CatalogContext
}
