package graph

import "sort"

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
