package walker

import (
	"github.com/fine-structures/fine-sdk-go/go2x3"
	"github.com/fine-structures/fine-sdk-go/lib2x3/graph"
)

// Primary entry point for "2x3" graph enumeration aka "fine structures"
func EnumPureParticles(opts EnumOpts) (*go2x3.GraphStream, error) {
	return enumPureParticles(opts)
}

// GrowOp is a graph building step, specifying how to add an vertex and/or edge
type GrowOp struct {
	OpCode graph.OpCode
	VtxA   graph.VtxID
	SlotA  uint8
}

func (op *GrowOp) EdgeSlotOrdinal() int {
	return int(op.VtxA)*graph.EdgesPerVertex + int(op.SlotA)
}

type EnumOpts struct {
	VertexMax int
	Params    string
	//Context go2x3.CatalogContext
}
