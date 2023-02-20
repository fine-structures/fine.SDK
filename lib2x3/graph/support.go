package graph

import (
	fmt "fmt"
	"sort"
	strconv "strconv"
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

func (X *VtxGraph) IsNormalized() bool {
	return X.Status >= VtxStatus_Canonized_Normalized
}

func (X *VtxGraph) IsCanonized() bool {
	return X.Status >= VtxStatus_Canonized
}


func (e *VtxEdge) Ord() int64 {
	return int64(e.DstVtxID) << 32 | int64(e.SrcVtxID) // sort by dst / "home" vtx ID first
}


func (e *VtxEdge) AppendDesc(io []byte) []byte {
	//var buf [24]byte
	
	str := fmt.Sprintf("    %c <=   +%02d-%02d <= %c ", 'A' - 1 + byte(e.DstVtxID), e.PosCount, e.NegCount, 'A' - 1 + byte(e.SrcVtxID))
	io = append(io, str...)
	
	// if strings.HasSuffix(str, "+0") {
	// 	io = append(io[:len(io)-2], ' ', ' ')
	// }
	
	
	// io = append(io, 'A' - 1 + byte(e.DstVtxID), ' ', '<', '=', ' ', ' ')
	// io = AppendInt02(io, +int64(e.PosCount))
	// io = AppendInt02(io, -int64(e.NegCount))

	
	// io = append(io, ' ', ' ', 'A' - 1 + byte(e.SrcVtxID), ' ', ' ', ' ')


	io = AppendInt02(io, int64(e.C1Seed))

	return io
}


func AppendInt02(dst []byte, val int64) []byte {
	if val == 0 {
		dst = append(dst, ' ', ' ', ' ')
		return dst
	}

	N := len(dst)
	dst = append(dst, '+')
	if val < 0 {
		val = -val
		dst[N] = '-'
	}
	
	if val <= 99 {
		dst = append(dst, '0', '0')
		next := val / 10
		digit := val - 10*next		
		dst[N+1] = '0' + byte(next)
		dst[N+2] = '0' + byte(digit)
	} else {
		dst = strconv.AppendInt(dst, val, 10)
	}
	return dst
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
	for j := i-1; j >= 0; j-- {
		dst[j] = ' '
	}
	return dst[i:]
}

