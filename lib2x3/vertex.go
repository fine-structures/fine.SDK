package lib2x3

import (
	"encoding/binary"
)

// VtxID is one-based index that identifies a vertex in a given graph (1..VtxMax)
type VtxID byte

// // VtxIdx is zero-based index that identifies a vertex in a given graph (0..VtxMax-1)
// type VtxIdx byte

const (

	// VtxMax is the max possible value of a VtxID (a one-based index).
	MaxVtxID = 36

	// VtxIDBits is the number of bits dedicated for a VtxID.  It must be enough bits to represent MaxVtxID.
	VtxIDBits byte = 6

	// VtxIDMask is the corresponding bit mask for a VtxID
	VtxIDMask VtxID = (1 << VtxIDBits) - 1

	// MaxEdgeEnds is the max number of possible edge connections for the largest graph possible.
	MaxEdges       = 3 * MaxVtxID / 2
	MaxEdgeEnds    = 3 * MaxVtxID
	MaxTraces      = MaxVtxID
	MaxTraceSpecSz = MaxVtxID * binary.MaxVarintLen64
)

// VtxCount signals a count of vertexes or edge slots
type VtxCount byte

// VtxType is one of the 10 fundamental 2x3 vertex types
type VtxType byte

const (
	V_nil   VtxType = 0
	V_e     VtxType = 1
	V_e_bar VtxType = 2
	V_π     VtxType = 3
	V_π_bar VtxType = 4
	V_u     VtxType = 5
	V_u_bar VtxType = 6
	V_q     VtxType = 7
	V_d     VtxType = 8
	V_d_bar VtxType = 9
	V_𝛾     VtxType = 10

	// VtxTypeMask masks the bits associated with VtxType
	VtxTypeMask VtxType = 0xF // 4 bits
)

var AllVtxTypes = [...]VtxType{
	V_e, V_e_bar,
	V_π, V_π_bar,
	V_u, V_u_bar, V_q,
	V_d, V_d_bar,
	V_𝛾,
}

func (v VtxType) Ord() byte {
	return byte(v)
}

func (v VtxType) String() string {
	return [...]string{"nil",
		"e", "~e",
		"π", "~π",
		"u", "~u", "q",
		"d", "~d",
		"y", // "𝛾"
	}[v]
}

func (v VtxType) NumEdges() byte {
	return [...]byte{0, 0, 0, 0, 0, 1, 1, 1, 2, 2, 3}[v]
}

func (v VtxType) PosLoops() byte {
	return [...]byte{0, 3, 0, 2, 1, 2, 0, 1, 1, 0, 0}[v]
}

func (v VtxType) NegLoops() byte {
	return [...]byte{0, 0, 3, 1, 2, 0, 2, 1, 0, 1, 0}[v]
}

func (v VtxType) TotalLoops() byte {
	return [...]byte{0, 3, 3, 3, 3, 2, 2, 2, 1, 1, 0}[v]
}

func (v VtxType) NetLoops() int8 {
	return [...]int8{0, 3, -3, 1, -1, 2, -2, 0, 1, -1, 0}[v]
}

func (v VtxType) VtxPerm() VtxPerm {
	return [...]VtxPerm{
		{},
		{4, [4]VtxType{V_e, V_π, V_π_bar, V_e_bar}}, // V_e
		{4, [4]VtxType{V_e_bar, V_e, V_π, V_π_bar}}, // V_e_bar
		{4, [4]VtxType{V_π, V_π_bar, V_e_bar, V_e}}, // V_π
		{4, [4]VtxType{V_π_bar, V_e_bar, V_e, V_π}}, // V_π_bar
		{3, [4]VtxType{V_u, V_q, V_u_bar}},          // V_u
		{3, [4]VtxType{V_q, V_u_bar, V_u}},          // V_q
		{3, [4]VtxType{V_u_bar, V_u, V_q}},          // V_u_bar
		{2, [4]VtxType{V_d, V_d_bar}},               // V_d
		{2, [4]VtxType{V_d_bar, V_d}},               // V_d_bar
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
