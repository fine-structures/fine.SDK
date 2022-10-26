package lib2x3

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
)

type (
	GroupID   byte
	Adjacency byte
)

const (
	GroupLoop GroupID = '*'
	GroupDupe GroupID = '|'
	GroupA    GroupID = 'A'

	Adjacent_0_3L Adjacency = 0 // loop,   loop    loop         v=1 (e, ~e, π, ~π)
	Adjacent_1_2L Adjacency = 1 // single  loop    loop         v=2 (single edge)
	Adjacent_1_1L Adjacency = 2 // double  loop    ---          v=2 (double edge)
	Adjacent_1_0L Adjacency = 3 // triple  ---     ---          v=2 (tri gamma)
	Adjacent_2_1L Adjacency = 4 // single  single  loop         v=3+
	Adjacent_2_0L Adjacency = 5 // single  double  ---          v=3+ (1-2 gamma)
	Adjacent_3_0L Adjacency = 6 // single  single  single       v=3+ (disjoint gamma)
)

type SignGroup struct {
	Signs OctSign
	Count byte
}

type Tricode struct {
	Src1       GroupID
	Src2       GroupID
	Src3       GroupID
	SignGroups []SignGroup // Subs?  Parts?
}

type TriID struct {
	NumGroups byte
	Tricodes  []Tricode
}

func (triID TriID) PrintString(maxV byte) {
	//
}

var OctSignStr = [1 + 8]string{
	" . ",
	"---", "--+", "-+-", "-++",
	"+--", "+-+", "++-", "+++",
}

type OctSign byte

const (
	OctSign_nil OctSign = 0x0
	OctSign_000 OctSign = 0x1
	OctSign_001 OctSign = 0x2
	OctSign_010 OctSign = 0x3
	OctSign_011 OctSign = 0x4
	OctSign_100 OctSign = 0x5
	OctSign_101 OctSign = 0x6
	OctSign_110 OctSign = 0x7
	OctSign_111 OctSign = 0x8
)

type TriGraphExpr struct {
	//Parts []*Part `(@@ (";" @@)*)?`
}

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
	vtx0      VtxID // VtxID of the current Part
	maxVtxID  VtxID
	vtxEdges  [MaxVtxID]byte
	vtxArrows [MaxVtxID]byte
	edges     []EdgeID
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
		return ErrGraphBadVtxID
	}
	if Xb.maxVtxID < vtxID {
		Xb.maxVtxID = vtxID
	}

	for _, r := range vtx.Kind {
		if r == '^' {
			Xb.vtxArrows[vtxID-1]++
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
			v := GetVtxType(Xb.vtxArrows[vi], Xb.vtxEdges[vi])
			if v == V_nil {
				vi_local := vi + 1 - Xb.vtx0
				return fmt.Errorf("error reading part #%d: arrows + edges at vertex %d exceeds 3", xi+1, vi_local)
			}
			X.vtx[X.vtxCount] = v
			X.vtxCount++
		}

		// TODO: validate that what was added was a single contiguous graph
	}

	// After we've absorbed all the edge parts, update X
	X.edgeCount = int32(len(Xb.edges))

	return nil
}
