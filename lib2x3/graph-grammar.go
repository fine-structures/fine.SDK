package lib2x3

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
)

// GroupID is a one-based index representing a group (vertex ordinality)  ID.
type GroupID byte

func (g GroupID) GroupRune() byte {
	if g > 0 {
		return 'A' - 1 + byte(g)
	}
	return '?'
}

// GroupEdge expresses edge embedding an inlet group number, edge sign, and if the edge is a loop.
type GroupEdge byte

func (e GroupEdge) EdgeTypeRune() byte {
	if e.IsLoop() {
		if e.IsNeg() {
			return '*'
		} else {
			return 'o'
		}
	} else if from := e.GroupID(); from > 0 {
		return '|'
	}
	return '?'
}

func (e GroupEdge) GroupRune() byte {
	return e.GroupID().GroupRune()
}

func (e GroupEdge) FromGroupRune() byte {
	if e.IsLoop() {
		return ' '
	}
	return e.GroupID().GroupRune()
}

func (e GroupEdge) SignRune() byte {
	if e.IsNeg() {
		return '-'
	}
	return '+'
}

func (e GroupEdge) IsLoop() bool {
	return int(e)&(1<<kLoopBitShift) != 0
}

// Returns 1 if loop, 0 if edge (non-loop).
func (e GroupEdge) LoopBit() int {
	return (int(e) >> kLoopBitShift) & 1
}

// Returns -1 or 1
func (e GroupEdge) Sign() int {
	return ((int(e) >> (kSignBitShift - 1)) & 0x2) - 1
}

func (e GroupEdge) IsNeg() bool {
	return int(e)&(1<<kSignBitShift) != 0
}

func (e GroupEdge) GroupID() GroupID {
	return GroupID(e) & kGroupID_Mask
}

func FormGroupEdge(i GroupID, isLoop, isNeg bool) GroupEdge {
	e := GroupEdge(i)
	if isLoop {
		e |= 1 << kLoopBitShift
	}
	if isNeg {
		e |= 1 << kSignBitShift
	}
	return e
}

const (
	GroupID_nil GroupID = 0

	kGroupID_Bits = 6
	kGroupID_Mask = (1 << kGroupID_Bits) - 1
	kLoopBitShift = 6 // When bit is set, edge is a loop
	kSignBitShift = 7 // When bit is set, edge sign is negative

)

const NumTriSigns = 8


type TriSign byte

func (t TriSign) String() string {
	return []string{
		"+++", "++-", "+-+", "+--",
		"-++", "-+-", "--+", "---",
	}[t]
}

type TriGroup struct {
	FamilyID GroupID           // which vtx family group this is
	CyclesID GroupID           // which cycles group this is
	Edges    [3]GroupEdge      // cycle group connections
	Counts   [NumTriSigns]int8 // instance counts
}

/*
// Returns sortable ordinal expressing the 3 bits in i={0,1,2} order:
//    Edges[0..2].IsLoop ? 0 : 1
func (g *TriGroup) EdgesType() VtxEdgesType {
	edges := int32(3)
	for _, ei := range g.Edges {
		edges -= ei.LoopBit()
	}
	return VtxEdgesType(edges)
}
*/

func (g *TriGroup) EdgesCycleOrd() int {
	ord := int(0)
	for _, ei := range g.Edges {
		ord = (ord << 8) | int(ei.GroupID())
	}
	return ord
}

func (g *TriGroup) VtxCount() int8 {
	var total int8
	for _, ci := range g.Counts {
		total += ci
	}
	return total
}

func (g *TriGroup) SignCardinality() int {
	numSigns := 0
	for _, ci := range g.Counts {
		if ci > 0 {
			numSigns++
		}
	}
	return numSigns
}

/*
func (g *TriGroup) Compare(src *TriGroup) int {
	if d := int(g.FamilyID) - int(src.FamilyID); d != 0 {
		return d
	}
	if d := int(g.EdgesType()) - int(src.EdgesType()); d != 0 {
		return d
	}
	if d := g.EdgesCycleOrd() - src.EdgesCycleOrd(); d != 0 {
		return d
	}
	if d := int(g.CyclesID) - int(src.CyclesID); d != 0 {
		return d
	}
	if d := g.SubGroupingOrd() - src.SubGroupingOrd(); d != 0 {
		return d
	}
	return 0
}



func (g *TriGroup) Consolidate(src *TriGroup) bool {
	if g.Compare(src) != 0 {
		return false
	}
	for i, ni := range src.Counts {
		g.Counts[i] += ni
	}
	return true
}


func (v* TriVtx) TriSign() TriSign {
	sign := byte(0)
	for i, ei := range v.edges {
		sign <<= 1
		if ei.edgeSign < 0 {
			sign |= 1
		}
	}
	return TriSign(sign)
}

*/
/*
type TriGroup struct {
	//This     GroupID

	Tricode [3]GroupID
	Triflag [3]
	Counts  [NumTriSigns]int32 // Run-length of each TriSign
}


func (tri *TriGroup) Ordinal() int32 {
	S1 := byte(tri.Src1)
	ord := int32(S1) << 16

	S2 := byte(tri.Src2)
	if tri.Src2 == GroupDupe {
		S2 = S1
		groupBits = 0x40
	}
	io = append(io, S2)

	S3 := byte(gi.Src3)
	if gi.Src3 == GroupDupe {
		S3 = S2
		groupBits = 0x0
	}
	io = append(io, S3)
}*/

func (g *TriGroup) AppendEdgesLabel(io []byte) []byte {
	for _, ei := range g.Edges {
		io = append(io, 'a'+byte(ei.GroupID())-1)
	}
	// for i, ei := range g.Edges {
	// 	c := byte('?')
	// 	// sign := g.
	// 	// if showTypes {
	// 	// 	switch {
	// 	// 		case ei.IsLoop && ei.Sign > 0:
	// 	// 			c = '*'
	// 	// 		case ei.IsLoop && ei.Sign < 0:
	// 	// 			c = 'o'
	// 	// 		case ei.Sign > 0:
	// 	// 			c = '+'
	// 	// 		case ei.Sign < 0:
	// 	// 			c = '-'
	// 	// 	}
	// 	// 	io = append(io, c)
	// 	// }
	// 	if ei.LoopBit() != 0 {
	// 		c = '*'
	// 	} else {
	// 		c = 'A' + ei.ToGroup - 1
	// 	}
	// 	switch {
	// 	case i > 0 && g.Grouping == Grouping_111:
	// 		c = '|'
	// 	case i == 1 && g.Grouping == Grouping_112:
	// 		c = '|'
	// 	}
	// 	io = append(io, c)

	return io
}

type TriGraph struct {
	//	This PhaseID // *trailing* byte lexicographically in encodings  (matter vs anti-matter)
	CGE    string // UTF-8 Conventional 2x3 Graph Encoding string expression
	Groups []TriGroup
}

func (X *TriGraph) Clear(numGroups int) {
	X.CGE = ""

	if cap(X.Groups) < numGroups {
		X.Groups = make([]TriGroup, numGroups)
	} else {
		X.Groups = X.Groups[:numGroups]
	}
	for i := range X.Groups {
		X.Groups[i] = TriGroup{
			//This: GroupA + GroupID(i),
		}
	}
}

func (X *TriGraph) VtxCount() int32 {
	Nv := int32(0)
	for _, gi := range X.Groups {
		for _, ci := range gi.Counts {
			Nv += int32(ci)
		}
	}
	return Nv
}

/*
func (X *TriGraph) Canonize() {
	sort.Slice(X.Groups, func(i, j int) bool {
		diff := X.Groups[i].Compare(&X.Groups[j])
		return diff < 0
	})
	X.Consolidate()
}

func (X *TriGraph) Consolidate() {

	// At this point, vertices are sorted via tricode (using group numbering via canonic ranking of cycle vectors)
	// Here we collapse consecutive vertices with the same tricode into a "super" group
	// As we collapse tricodes, we must reassign new groupIDs
	{ //for running := true; running; {
		Xg := X.Groups
		Ng := len(Xg)
		L := byte(0)
		for R := 1; R < Ng; R++ {
			XgL := &Xg[L]
			XgR := &Xg[R]
			if !XgL.Consolidate(XgR) {
				L++
				Xg[L] = *XgR
			}
		}
		X.Groups = X.Groups[:L+1]
	}
}
*/

// Note that all Encodings have an implied "anti-matter" phase, which just flips all the TriSigns.
type TriGraphEncoderOpts int

const (
	IncludeSignModes TriGraphEncoderOpts = 1 << iota

	//TracesAndModesAndCGE
)

func (X *TriGraph) ExportGraphDesc(io []byte) []byte {
	//b := strings.Builder{}

	//var scrap [32]byte

	// Gn - Group count
	//fmt.Fprintf(&buf, "%d:", len(X.Groups))
	//io = append(io, byte('0' + len(X.Groups)))

	//
	for i, gi := range X.Groups {
		//b.Reset()
		if i > 0 {
			io = append(io, ',')
		}
		numSigns := gi.SignCardinality()
		if numSigns > 1 {
			io = append(io, '(')
		}
		first := true
		for i, ci := range gi.Counts {
			if ci > 0 {
				if !first {
					first = false
					io = append(io, ' ')
				}
				{
					if ci >= 10 {
						io = append(io, '0'+byte(ci/10))
					}
					io = append(io, '0'+byte(ci%10))
				}
				io = append(io, TriSign(i).String()...)
			}
		}
		if numSigns > 1 {
			io = append(io, ')')
		}

		io = gi.AppendEdgesLabel(io)

		//io = append(io, b.String()...)
	}

	return io
}

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
