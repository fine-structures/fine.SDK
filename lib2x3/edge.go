package lib2x3

import (
	"sort"

	"github.com/2x3systems/go2x3/go2x3"
)

// EdgeID contains a two VtxIDs and an EdgeType
// (Va <<) | (Vb <<) | (EdgeType), where Va < Vb
type EdgeID uint16

// EdgeType names the type of edge
type EdgeType byte

const (
	NilEdge       EdgeType = 0
	PosEdge       EdgeType = (1 << 2)
	NegEdge       EdgeType = (1 << 2) | 1
	PosPosEdge    EdgeType = (2 << 2)
	PosNegEdge    EdgeType = (2 << 2) | 1
	NegNegEdge    EdgeType = (2 << 2) | 2
	PosPosPosEdge EdgeType = (3 << 2)
	PosPosNegEdge EdgeType = (3 << 2) | 1
	PosNegNegEdge EdgeType = (3 << 2) | 2
	NegNegNegEdge EdgeType = (3 << 2) | 3

	EdgeTypeToEdgeIDShift byte   = 12
	EdgeTypeMask          EdgeID = 0xF000 // high 4 bits
)

func (et EdgeType) TotalEdges() byte {
	return byte(et>>2) & 0x3
}

func (et EdgeType) NumPosNeg() (numPos, numNeg byte) {
	numNeg = byte(et & 0x3)
	numPos = et.TotalEdges() - numNeg
	return
}

func (et EdgeType) CombineWith(other EdgeType) EdgeType {
	neg := byte(et&0x3) + byte(other&0x3)
	total := et.TotalEdges() + other.TotalEdges()

	return EdgeType((total << 2) | neg)
}

func (et EdgeType) EdgeSum() int32 {
	total := int32(et>>2) & 0x3
	neg := int32(et & 0x3)
	return total - 2*neg
}

func (et EdgeType) String() string {
	return [16]string{
		" ", " ", " ", " ",
		"-", "~", " ", " ",
		"=", "-~", "~~", " ",
		"---", "--~", "-~~", "~~~",
	}[et]
}

// FormEdge forms a canonical EdgeID with the given EdgeType, Va, and Vb.
func (et EdgeType) FormEdge(Va, Vb VtxID) EdgeID {
	var edge EdgeID
	if Va < Vb {
		edge = (EdgeID(Va) << 6) | EdgeID(Vb)
	} else {
		edge = (EdgeID(Vb) << 6) | EdgeID(Va)
	}
	if Va < 1 || Vb < 1 {
		panic("invalid VtxIDs given to FromEdge()")
	}
	edge |= EdgeID(et) << EdgeTypeToEdgeIDShift
	return edge
}

func parseEdgeStr(str string) (EdgeType, int, error) {
	pos := byte(0)
	neg := byte(0)
	i := 0
	for _, r := range str {
		switch r {
		case '-':
			pos += 1
		case '~':
			neg += 1

		case '=':
			pos += 2
		case '≃':
			pos += 1
			neg += 1
		case '≈':
			neg += 2

		case '≡':
			pos += 3
		case '≅':
			neg += 1
			pos += 2
		case '≊':
			neg += 2
			pos += 1
		case '≋':
			neg += 3

		default:
			if i == 0 {
				return NilEdge, 0, nil
			}
		}
		i++
	}
	total := pos + neg
	if pos+neg > 3 {
		return NilEdge, i, go2x3.ErrBadEdgeType
	}
	edgeType := EdgeType((total << 2) | neg)
	return edgeType, i, nil
}

func (edge EdgeID) EdgeType() EdgeType {
	return EdgeType(edge >> EdgeTypeToEdgeIDShift)
}

func (edge EdgeID) EdgeSum() int32 {
	return edge.EdgeType().EdgeSum()
}

func (edge EdgeID) VtxAB() (a, b VtxID) {
	a = VtxID(edge>>6) & VtxIDMask
	b = VtxID(edge>>0) & VtxIDMask
	return
}

func (edge EdgeID) VtxIdx() (a, b int) {
	a = int((VtxID(edge>>6) & VtxIDMask) - 1)
	b = int((VtxID(edge>>0) & VtxIDMask) - 1)
	return
}

func (edge EdgeID) ChangeEdgeType(et EdgeType) EdgeID {
	ab := edge &^ EdgeTypeMask
	return (EdgeID(et) << EdgeTypeToEdgeIDShift) | ab
}

func (edge EdgeID) EdgePerm() EdgePerm {
	et := edge.EdgeType()
	totalEdges := int32(et.TotalEdges())

	// As it turns out, the number of combos for num edges = 1,2,3 happens to be 2,3,4
	perm := EdgePerm{
		Num: totalEdges + 1,
	}

	Va, Vb := edge.VtxAB()

	// Start at the given EdgeType
	neg := int32(et & 0x3)
	for i := int32(0); i < perm.Num; i++ {
		et_i := EdgeType(totalEdges<<2 | neg)
		perm.Edges[i] = et_i.FormEdge(Va, Vb)

		// Wrap when at the permutation limit
		neg++
		if neg == perm.Num {
			neg = 0
		}
	}
	return perm
}

type EdgePerm struct {
	Num   int32
	Edges [4]EdgeID
}

type EdgeList []EdgeID

func (es EdgeList) Len() int           { return len(es) }
func (es EdgeList) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es EdgeList) Less(i, j int) bool { return es[i] < es[j] }

func (es EdgeList) Canonicalize() {
	sort.Sort(es)
}

// SwapVtxID swaps all instances of vi and vj in this this edge list (and is used when relabeling verticies).
func (es EdgeList) SwapVtxID(vi, vj VtxID) {
	for i, edge := range es {
		a, b := edge.VtxAB()

		// If one of the VtxIDs matches what we need to swap, do the swap and rewrite the edge
		if a == vi {
			a = vj
		} else if b == vi {
			b = vj
		} else if a == vj {
			a = vi
		} else if b == vj {
			b = vi
		} else {
			continue
		}

		es[i] = edge.EdgeType().FormEdge(a, b)
	}
}

// ShortestEdgeDist returns the shortest adjacency distance of two vertices at the given 0-based index.
func ShortestEdgeDist(Nv, i, j int32) int32 {
	dist := i - j
	if dist < 0 {
		dist = -dist
	}
	flipAt := Nv >> 1
	if dist > flipAt {
		dist = Nv - dist
	}
	return dist
}
