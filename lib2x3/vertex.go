package lib2x3

import (
	"github.com/fine-structures/fine-sdk-go/go2x3"
)

// VtxID is one-based index that identifies a vertex in a given graph (1..VtxMax)
type VtxID byte

// // VtxIdx is zero-based index that identifies a vertex in a given graph (0..VtxMax-1)
// type VtxIdx byte

const (

	// VtxMax is the max possible value of a VtxID (a one-based index).
	MaxVtxID = go2x3.MaxVtxID
	
	// VtxIDBits is the number of bits dedicated for a VtxID.  It must be enough bits to represent MaxVtxID.
	VtxIDBits byte = 5

	// VtxIDMask is the corresponding bit mask for a VtxID
	VtxIDMask VtxID = (1 << VtxIDBits) - 1

	MaxEdges = go2x3.MaxEdges
)

// VtxCount signals a count of vertexes or edge slots
type VtxCount byte

// VtxType is one of the 10 fundamental 2x3 vertex types
type VtxType byte

const (
	V_nil VtxType = 0

	V_e_bar VtxType = 1  // ***
	V_π_bar VtxType = 2  // **o
	V_π     VtxType = 3  // *oo
	V_e     VtxType = 4  // ooo
	V_u_bar VtxType = 5  // **|
	V_q     VtxType = 6  // *o|
	V_u     VtxType = 7  // oo|
	V_d_bar VtxType = 8  // *||
	V_d     VtxType = 9  // o||
	V_𝛾     VtxType = 10 // |||

	// VtxTypeMask masks the bits associated with VtxType
	VtxTypeMask VtxType = 0xF // 4 bits
)

func (v VtxType) Ord() byte {
	return byte(v)
}

func (v VtxType) String() string {
	return [...]string{"nil",
		"~e", "~π", "π", "e",
		"~u", "q", "u",
		"~d", "d",
		"y", // "𝛾"
	}[v]
}

func (v VtxType) NumEdges() byte {
	return [...]byte{0, 0, 0, 0, 0, 1, 1, 1, 2, 2, 3}[v]
}

func (v VtxType) PosLoops() byte {
	return [...]byte{0, 0, 1, 2, 3, 0, 1, 2, 0, 1, 0}[v]
}

func (v VtxType) NegLoops() byte {
	return [...]byte{0, 3, 2, 1, 0, 2, 1, 0, 1, 0, 0}[v]
}

func (v VtxType) NetLoops() int8 {
	return [...]int8{0, -3, -1, 1, 3, -2, 0, 2, -1, 1, 0}[v]
}

/*
func (v VtxType) SelfEdgeType() EdgeType {
	return [...]EdgeType{
		NegNegNegEdge,
		PosNegNegEdge,
		PosPosNegEdge,
		PosPosPosEdge,
		NegNegEdge,
		PosNegEdge,
		PosPosEdge,
		NegEdge,
		PosEdge,
		NilEdge,
	}[v]
}
*/

func (v VtxType) VtxPerm() VtxPerm {
	return [...]VtxPerm{
		{},
		{4, [4]VtxType{V_e_bar, V_π_bar, V_π, V_e}}, // V_e_bar
		{4, [4]VtxType{V_π_bar, V_π, V_e, V_e_bar}}, // V_π_bar
		{4, [4]VtxType{V_π, V_e, V_e_bar, V_π_bar}}, // V_π
		{4, [4]VtxType{V_e, V_π, V_π_bar, V_e_bar}}, // V_e
		{3, [4]VtxType{V_u_bar, V_q, V_u}},          // V_u_bar
		{3, [4]VtxType{V_q, V_u, V_u_bar}},          // V_q
		{3, [4]VtxType{V_u, V_q, V_u_bar}},          // V_u
		{2, [4]VtxType{V_d_bar, V_d}},               // V_d_bar
		{2, [4]VtxType{V_d, V_d_bar}},               // V_d
		{1, [4]VtxType{V_𝛾}},                        // V_𝛾
	}[v]
}

type VtxPerm struct {
	Num int32
	Vtx [4]VtxType
}

// GetVtxType returns the VtxType that corresponds to the given number of negative loops and total edges.
//
// If numNegLoops or numEdges are invalid, V_nil is returned
func GetVtxType(numNegLoops, numEdges byte) VtxType {
	v := V_nil
	switch numEdges {
	case 0:
		switch numNegLoops {
		case 0:
			v = V_e
		case 1:
			v = V_π
		case 2:
			v = V_π_bar
		case 3:
			v = V_e_bar
		}
	case 1:
		switch numNegLoops {
		case 0:
			v = V_u
		case 1:
			v = V_q
		case 2:
			v = V_u_bar
		}
	case 2:
		switch numNegLoops {
		case 0:
			v = V_d
		case 1:
			v = V_d_bar
		}
	case 3:
		switch numNegLoops {
		case 0:
			v = V_𝛾
		}
	}
	return v
}
