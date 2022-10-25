package graph

func (Xdef *GraphDef) AssignCopy(src *GraphDef) {

	buf := Xdef.GraphEncoding[:0]
	if src == nil {
		Xdef.Reset()
	} else {
		*Xdef = *src
	}

	// Reuse allocs
	Xdef.GraphEncoding = buf
}

func (Xdef *GraphDef) Clear() {
	Xdef.AssignCopy(nil)
}
