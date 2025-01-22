package graph

import (
	"container/heap"
	"sync/atomic"

	"github.com/art-media-platform/amp.SDK/stdlib/symbol"
	"github.com/art-media-platform/amp.SDK/stdlib/symbol/memory_table"
	"github.com/fine-structures/fine.SDK/go2x3"
)

type walker struct {
	EnumStream *go2x3.GraphStream
	forkCount  atomic.Uint64
	opts       EnumOpts
	emitted    symbol.Table
	emitQueue  Heap
}

func enumPureParticles(opts *EnumOpts) (*go2x3.GraphStream, error) {
	if opts.Context == nil {
		opts.Context = go2x3.NewCatalogContext()
	}
	//ctx := go2x3.NewCatalogContext() TODO?
	tableOpts := memory_table.DefaultOpts()
	emitted, err := tableOpts.CreateTable()
	if err != nil {
		return nil, err
	}

	w := &walker{
		opts:    *opts,
		emitted: emitted,
		EnumStream: &go2x3.GraphStream{
			Outlet: make(chan go2x3.State, 1),
		},
	}

	// seed with a single "zero" vertex state
	X := NewState(nil)
	w.emitQueue.Push(X)

	// gopher it!
	go func() {
		w.go_emit()
	}()

	return w.EnumStream, nil
}

type Heap struct {
	m []*state
}

func (h *Heap) Len() int { return len(h.m) }
func (h *Heap) Less(i, j int) bool {
	mi := h.m[i].AssemblyMetric()
	mj := h.m[j].AssemblyMetric()
	return mi < mj
}
func (h *Heap) Swap(i, j int) {
	h.m[i], h.m[j] = h.m[j], h.m[i]
}

// Pushes a new state onto the heap
func (h *Heap) Push(X any) {
	h.m = append(h.m, X.(*state))
}

// Pops the top state from the heap
func (h *Heap) Pop() any {
	N := len(h.m) - 1
	item := h.m[N]
	h.m = h.m[0:N]
	return item
}

func (h *Heap) Peek() *state {
	if h.Len() == 0 {
		return nil
	}
	return h.m[0]
}

func (h *Heap) Insert(X *state) {
	idx := h.Len()
	h.Push(X)
	heap.Fix(h, idx)
}
