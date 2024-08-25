package walker

import (
	"github.com/astronomical-grace/fine-structures-go/go2x3"
	"github.com/astronomical-grace/fine-structures-go/lib2x3/graph"
)

// Primary entry point for "2x3" graph enumeration aka "fine structures"
func EnumPureParticles(opts EnumOpts) (*go2x3.GraphStream, error) {
	return enumPureParticles(opts)
}

// GrowOp is a graph building step, specifying how to add an vertex and/or edge
type GrowOp struct {
	OpCode graph.OpCode
	VtxA   VtxID
	SlotA  uint8
}

// func (op *GrowOp) AddsVertex() bool {
// 	return op.OpCode == graph.OpAddVertex || op.OpCode == graph.OpAddVertexAndEdge
// }

func (op *GrowOp) EdgeSlotOrdinal() int {
	return int(op.VtxA)*EdgesPerVertex + int(op.SlotA)
}

type EdgeSlot struct {
	To   VtxID // 1, 2, 3, ..   where 0 denotes inward / unassigned edge
	Sign Sign  // +1, 0, -1
}

// Vertex is a node of a graph, with a fixed number of edges per vertex
type Vertex struct {
	ID    VtxID // 1, 2, 3, ..
	Count int64 // aka accumulation
	Edges [EdgesPerVertex]EdgeSlot
}

// VtxID is one-based index that identifies a vertex in a given graph (1..VtxMax)
type VtxID byte

// SlotID embeds a one-based vertex index (1..VtxMax) and zero-based slot index (0..EdgesPerVertex-1)
type EdgeSlotID byte

func (slot EdgeSlotID) VertexAndSlot() (vertexIdx, slotIndex uint8) {
	return uint8(slot) / EdgesPerVertex, uint8(slot) % EdgesPerVertex
}

type EnumOpts struct {
	VertexMax int
	Params    string
	//Context go2x3.CatalogContext
}

// Sign specifies an edge sign or weight
type Sign byte

const (
	EdgesPerVertex = 3

	Sign_Natural Sign = 0 // odd cycles are not inverted (default)
	Sign_Invert  Sign = 1 // odd cycles are inverted
	Sign_Zero    Sign = 2 // odd cycles are 0
)
