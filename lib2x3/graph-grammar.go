package lib2x3

import (
	"fmt"

	"github.com/2x3systems/go2x3/lib2x3/graph"
	"github.com/alecthomas/participle/v2"
)

type GraphExpr struct {
	Parts []*Part `(@@ (";" @@)*)?`
}

type Part struct {
	EdgeRuns []*EdgeRun `(@@ ("," @@)*)?`
}

type EdgeRun struct {
	StartVtx *Vtx       `@@`
	Edges    []*EdgeDst `@@*`
}

type EdgeDst struct {
	Kind   string `@( "-" "~"? "-"? "~"? "-"? | "~" "-"? "~"? "-"? "~"? | "=" )`
	EndVtx *Vtx   `@@`
}

type Vtx struct {
	ID   int64  `@Int`
	Kind string `@( "^"* )`
}

type graphBuilder struct {
	vtx0        VtxID // VtxID of the current Part
	maxVtxID    VtxID
	vtxEdges    [MaxVtxID]byte
	vtxNegLoops [MaxVtxID]byte
	edges       []EdgeID
}

func (Xb *graphBuilder) applyPart(part *Part) error {
	Xb.vtx0 = Xb.maxVtxID

	for _, run := range part.EdgeRuns {
		err := Xb.applyRun(run)
		if err != nil {
			return err
		}
	}

	return nil
}

func (Xb *graphBuilder) tallyVtx(vtx *Vtx) error {
	vtxID := Xb.vtx0 + VtxID(vtx.ID)

	if vtxID < 1 || vtxID > MaxVtxID {
		return graph.ErrBadVtxID
	}
	if Xb.maxVtxID < vtxID {
		Xb.maxVtxID = vtxID
	}

	for _, r := range vtx.Kind {
		if r == '^' {
			Xb.vtxNegLoops[vtxID-1]++
		}
	}

	return nil
}

func (Xb *graphBuilder) applyRun(run *EdgeRun) error {
	onVtx := run.StartVtx
	if err := Xb.tallyVtx(onVtx); err != nil {
		return err
	}

	for _, edge := range run.Edges {
		edgeType, _, err := parseEdgeStr(edge.Kind)
		if err != nil {
			return err
		}

		nextVtx := edge.EndVtx
		if err := Xb.tallyVtx(nextVtx); err != nil {
			return err
		}

		curID := Xb.vtx0 + VtxID(onVtx.ID)
		nxtID := Xb.vtx0 + VtxID(nextVtx.ID)
		Xb.edges = append(Xb.edges, edgeType.FormEdge(curID, nxtID))

		Xb.vtxEdges[curID-1] += edgeType.TotalEdges()
		Xb.vtxEdges[nxtID-1] += edgeType.TotalEdges()

		onVtx = nextVtx
	}

	return nil
}

var parseGraphExpr = participle.MustBuild[GraphExpr]() //, participle.UseLookahead(2))

func (X *Graph) InitFromString(graphExpr string) error {
	X.Init(nil)

	Xexpr, err := parseGraphExpr.ParseString("", graphExpr)
	if err != nil {
		return err
	}

	var Xb graphBuilder
	Xb.edges = X.edges[:0]

	for xi, part := range Xexpr.Parts {
		err = Xb.applyPart(part)
		if err != nil {
			return err
		}

		// Determine vertex types and validate what we can
		for vi := Xb.vtx0; vi < Xb.maxVtxID; vi++ {
			v := GetVtxType(Xb.vtxNegLoops[vi], Xb.vtxEdges[vi])
			if v == V_nil {
				vi_local := vi + 1 - Xb.vtx0
				return fmt.Errorf("error reading part #%d: loops + edges at vertex %d exceeds 3", xi+1, vi_local)
			}
			X.vtx[X.vtxCount] = v
			X.vtxCount++
		}

		// TODO: validate that what was added was a single contiguous graph
	}

	// After we've absorbed all the edge parts, update X
	X.edgeCount = len(Xb.edges)
	X.Def.TryAddGraphExpr(graphExpr)
	return nil
}
