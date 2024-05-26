package graph

import (
	"fmt"
	"sort"
)

// Adds the given graph expr string to .GraphExprs[] if it is not already present.
// Returns true if the string was added.
// Pre + Post: GraphExprs[] is sorted.
func (def *GraphDef) TryAddGraphExpr(graphExprStr string) bool {

	// If duplicate, no-op and return false
	idx := sort.SearchStrings(def.GraphExprs, graphExprStr)
	if idx < len(def.GraphExprs) && def.GraphExprs[idx] == graphExprStr {
		return false
	}

	N := len(def.GraphExprs)
	if cap(def.GraphExprs) == N {
		capSz := 2 * N
		if capSz < 8 {
			capSz = 8
		}
		newBuf := make([]string, N, capSz)
		copy(newBuf, def.GraphExprs)
		def.GraphExprs = newBuf
	}

	def.GraphExprs = def.GraphExprs[:N+1]
	if idx < N {
		copy(def.GraphExprs[idx+1:], def.GraphExprs[idx:])
	}
	def.GraphExprs[idx] = graphExprStr
	return true
}

func (def *GraphDef) AssignFrom(src *GraphDef) {
	encBuf := def.GraphEncoding[:0]
	exprs := def.GraphExprs[:0]

	// Reuse allocs
	if src == nil {
		*def = GraphDef{}
		def.GraphEncoding = encBuf
		def.GraphExprs = exprs
	} else {
		*def = *src
		def.GraphEncoding = append(encBuf, src.GraphEncoding...)
		def.GraphExprs = append(exprs, src.GraphExprs...)
	}
}

func FormGroupID(groupNum int32) GroupID {
	return GroupID_G1 + GroupID(groupNum-1)
}

func (g GroupID) GroupRune() byte {
	r := byte('.')
	switch {
	case g == GroupID_LoopVtx:
		r = 'o'
	case g == GroupID_LoopGroup:
		r = 'O'
	case g >= GroupID_G1:
		r = 'A' + byte(g-GroupID_G1)
	}
	return r
}

func (X *VtxGraph) IsComputed() bool {
	return X.Status >= GraphStatus_Computed
}

func (X *VtxGraph) IsCanonized() bool {
	return X.Status >= GraphStatus_Canonized
}

var signChar = [3]byte{' ', ' ', '~'}

func (v *VtxGroup) AppendDesc(io []byte) []byte {
	if v.GroupID == 1 {
		io = fmt.Appendf(io, " %2d", v.GraphID)
	} else {
		io = fmt.Append(io, "   ")
	}

	c := byte('?')
	if v.GroupID > 0 {
		c = 'A' + byte(v.GroupID) - 1
	}
	io = fmt.Appendf(io, ".%c           %c%02d", c, signChar[v.OddSign], v.Count)

	return io
}

type PrintIntOpts struct {
	MinWidth   int
	AlwaysSign bool
	SpaceZero  bool
}

func AppendInt(io []byte, val int64, opts PrintIntOpts) []byte {
	var digits [24]byte
	sgn := byte(0)
	if val < 0 {
		sgn = '-'
		val = -val
	} else if opts.AlwaysSign && val > 0 {
		sgn = '+'
	}

	N := 0
	for {
		next := val / 10
		digit := val - 10*next
		val = next
		digits[N] = '0' + byte(digit)
		N++
		if val == 0 {
			break
		}
	}
	if sgn != 0 {
		digits[N] = sgn
		N++
	}

	for i := N; i < opts.MinWidth; i++ {
		io = append(io, ' ')
	}
	if opts.SpaceZero && N == 1 && digits[0] == '0' {
		io = append(io, ' ')
	} else {
		for i := N - 1; i >= 0; i-- {
			io = append(io, digits[i])
		}
	}

	return io
}

// PrintInt prints the given integer in base 10, right justified in the buffer.
// Returns the tight-fitting slice of the output digits (a slice of []dst)
func PrintInt(dst []byte, val int64) []byte {
	sign := int(1)
	if val < 0 {
		sign = -1
		val = -val
	}
	L := len(dst)
	i := L
	for {
		next := val / 10
		digit := val - 10*next
		val = next
		i--
		dst[i] = '0' + byte(digit)
		if val == 0 {
			break
		}
	}
	if sign < 0 {
		i--
		dst[i] = '-'
	}
	for j := i - 1; j >= 0; j-- {
		dst[j] = ' '
	}
	return dst[i:]
}

func chopBuf(consume []int64, N int) (alloc []int64, remain []int64) {
	return consume[0:N], consume[N:]
}
