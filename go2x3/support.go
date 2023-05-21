package go2x3

import "sync"

func (factors *FactorSet) Insert(toAdd TracesID) {
	insertAt := len(*factors)

	for i, Fi := range *factors {
		if Fi.ID == toAdd {
			(*factors)[i].Count++
			return
		} else if Fi.ID > toAdd {
			insertAt = i
			break
		}
	}

	fax := append((*factors), FactorRun{})
	N := len(fax)
	copy(fax[insertAt+1:N], fax[insertAt:N-1])
	fax[insertAt] = FactorRun{
		ID:    toAdd,
		Count: 1,
	}
	*factors = fax
}

func FactorSetComparator(A, B FactorSet) int {
	lenB := len(B)

	for i, ai := range A {
		if lenB == i {
			return 1
		}

		bi := B[i]
		dID := int(ai.ID) - int(bi.ID)
		if dID != 0 {
			return dID
		}
		dCount := int(ai.Count) - int(bi.Count)
		if dCount != 0 {
			return dCount
		}
	}

	if len(A) > lenB {
		return -1
	}

	return 0
}

func (factors *FactorSet) Clear() {
	*factors = (*factors)[:0]
}

func (factors FactorSet) TotalVtxCount() uint32 {
	Nv := uint32(0)
	for _, Fi := range factors {
		Nv += Fi.Count * Fi.ID.NumVertices()
	}
	return Nv
}

func NewCatalogContext() CatalogContext {
	ctx := &catalogContext{
		openCatalogs: make(map[Catalog]struct{}),
		closing:      make(chan struct{}),
		closed:       make(chan struct{}),
	}
	ctx.openCount.Add(1)
	go func() {
		<-ctx.Closing()
		ctx.openCount.Done()
		ctx.openCount.Wait()
		close(ctx.closed)
	}()
	return ctx
}

type catalogContext struct {
	mu           sync.Mutex
	openCount    sync.WaitGroup
	openCatalogs map[Catalog]struct{}
	closing      chan struct{}
	closed       chan struct{}
}

func (ctx *catalogContext) AttachCatalog(cat Catalog) {
	ctx.openCount.Add(1)
	ctx.mu.Lock()
	ctx.openCatalogs[cat] = struct{}{}
	ctx.mu.Unlock()
}

func (ctx *catalogContext) DetachCatalog(cat Catalog) {
	ctx.mu.Lock()
	if _, exists := ctx.openCatalogs[cat]; exists {
		delete(ctx.openCatalogs, cat)
		ctx.openCount.Done()
	}
	ctx.mu.Unlock()
}

func (ctx *catalogContext) Closing() <-chan struct{} {
	return ctx.closing
}

func (ctx *catalogContext) Done() <-chan struct{} {
	return ctx.closed
}

func (ctx *catalogContext) Close() {
	close(ctx.closing)
	ctx.mu.Lock()
	for cat := range ctx.openCatalogs {
		go cat.Close()
	}
	ctx.mu.Unlock()

}




// Appends the defining info about a Graph to the given buffer.
//
// The order of fields is such that, lexicographically, graphs with all-positive edges and vertices appear first,
// then graphs with only with all-positive edges, then all other graphs.
func (info *GraphInfo) AppendGraphEncodingHeader(prefix []byte) []byte {
	prefix = append(prefix,
		info.NumParticles,
		info.NumVerts,
		info.NegEdges,
		info.NegLoops,
		info.PosLoops,
	)
	return prefix
}

// NumEdges returns the number of edges implied for a graph that has a given number of vertices and total loop count.
func (info *GraphInfo) NumEdges() byte {
	return (3*info.NumVerts - info.PosLoops - info.NegLoops) / 2

}


// DefaultGraphSelector selects all valid lib2x3 graphs.
var DefaultGraphSelector = GraphSelector{
	Min: GraphInfo{
		NumParticles: 1,
		NumVerts:     1,
	},
	Max: GraphInfo{
		NumParticles: MaxVtxID,
		NumVerts:     MaxVtxID,
		NegEdges:     MaxEdges,
		PosEdges:     MaxEdges,
		PosLoops:     3 * MaxVtxID,
		NegLoops:     3 * MaxVtxID,
	},
}

// AllowGraph is a convenience function used to see if a Graph is selected according to a GraphSelector.
func (sel *GraphSelector) SelectsGraph(X GraphState) bool {
	info := X.GetInfo()
	if info.NumParticles < sel.Min.NumParticles || info.NumVerts < sel.Min.NumVerts || info.PosLoops < sel.Min.PosLoops || info.NegLoops < sel.Min.NegLoops || info.PosEdges < sel.Min.PosEdges || info.NegEdges < sel.Min.NegEdges {
		return false
	}
	if info.NumParticles > sel.Max.NumParticles || info.NumVerts > sel.Max.NumVerts || info.PosLoops > sel.Max.PosLoops || info.NegLoops > sel.Max.NegLoops || info.PosEdges > sel.Max.PosEdges || info.NegEdges > sel.Max.NegEdges {
		return false
	}
	return true
}

