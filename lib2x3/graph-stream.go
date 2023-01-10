package lib2x3

import (
	"fmt"
	"io"
	"log"
	"strings"
)

type GraphAdder interface {
	TryAddGraph(X *Graph) bool
	Close()
}

type AddGraphOpts struct {
	AutoCloseCatalog bool
}

type GraphStream struct {
	Outlet chan *Graph
}

func EnumPureParticles(v_min, v_max int, method string) *GraphStream {
	gw, err := NewGraphWalker()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		gw.EnumPureParticles(v_min, v_max)
	}()

	return gw.EnumStream
}

func NewGraphStream() *GraphStream {
	stream := &GraphStream{
		Outlet: make(chan *Graph),
	}
	return stream
}

func StreamGraph(X *Graph) *GraphStream {
	next := NewGraphStream()

	go func() {
		next.Outlet <- NewGraph(X)
		next.Close()
	}()

	return next
}

func (stream *GraphStream) Close() {
	if stream.Outlet != nil {
		close(stream.Outlet)
	}
}

func (stream *GraphStream) PushGraph(X *Graph) {
	stream.Outlet <- NewGraph(X)
}

func (stream *GraphStream) PullGraph() *Graph {
	X := <-stream.Outlet
	return X
}

func (stream *GraphStream) PullAll() int {
	count := int(0)
	for X := range stream.Outlet {
		count++
		X.Reclaim()
	}
	return count
}

func (stream *GraphStream) Print(
	out io.WriteCloser,
	opts PrintOpts) *GraphStream {

	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		buf := strings.Builder{}
		buf.Grow(256)

		count := 0
		for X := range stream.Outlet {
			if len(opts.Label) > 0 {
				buf.WriteString(opts.Label)
			}
			buf.WriteByte(',')

			count++
			fmt.Fprintf(&buf, "%06d,", count)
			X.WriteAsString(&buf, opts)
			buf.WriteByte('\n')
			out.Write([]byte(buf.String()))
			buf.Reset()
			next.Outlet <- X
		}
		out.Close()
		next.Close()
	}()

	return next
}

func (stream *GraphStream) AddTo(target GraphAdder, opts AddGraphOpts) *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		for X := range stream.Outlet {
			wasAdded := target.TryAddGraph(X)
			if wasAdded {
				next.Outlet <- X
			} else {
				X.Reclaim()
			}
		}
		if opts.AutoCloseCatalog {
			target.Close()
		}
		next.Close()
	}()

	return next
}

func SelectFromCatalog(cat Catalog, sel GraphSelector) *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	onHit := make(chan *Graph, 4)

	go func() {
		cat.Select(sel, onHit)
		close(onHit)
	}()

	go func() {
		for X := range onHit {
			if sel.AllowGraph(X) {
				next.Outlet <- X
			} else {
				X.Reclaim()
			}
		}
		next.Close()
	}()

	return next
}

func (stream *GraphStream) SelectFromStream(sel GraphSelector) *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		var matchTraces Traces
		if sel.Traces != nil {
			matchTraces = sel.Traces.Traces(0)
		}
		for X := range stream.Outlet {
			keep := false
			if sel.AllowGraph(X) {
				keep = true
				if len(matchTraces) > 0 {
					TX := X.Traces(len(matchTraces))
					keep = matchTraces.IsEqual(TX)
				}
			}
			if keep {
				next.Outlet <- X
			} else {
				X.Reclaim()
			}
		}
		next.Close()
	}()

	return next
}

func (stream *GraphStream) Canonize(normalize bool) *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		for X := range stream.Outlet {
			err := X.Canonize(normalize)
			if err != nil {
				panic(err)
			}
			next.Outlet <- X
		}
		next.Close()
	}()

	return next
}

/*
type canonizeCtx struct {
	Gin  orca.GraphIn
	Gout orca.GraphOut
	gc   orca.IGraphCanonizer
}

func newCanonizeCtx() *canonizeCtx {
	Gin, Gout := orca.NewGraphIO()
	return &canonizeCtx{
		Gin:  Gin,
		Gout: Gout,
		gc:   orca.NewCanonizer(orca.DefaultCanonizerOpts),
	}
}

func (ctx *canonizeCtx) goCanonize(X *Graph) error {

	{
		Gout := ctx.Gout

		// Send X's vertices and edges to Gout
		go func() {

			for i, v := range X.Vtx() {
				Gout.Vtx <- orca.Vtx{
					Color: orca.VtxColor(v),
					Label: orca.VtxLabel(i + 1),
				}
			}

			for _, e := range X.Edges() {
				Gout.Edges <- e.OrcaEdge()
			}
			Gout.Break()
		}()

		if err := ctx.gc.BuildGraph(ctx.Gin); err != nil {
			return err
		}

		go ctx.gc.Canonize(Gout)
	}

	{
		// var t1, t2 [128]byte
		// T1 := X.Traces(0).AppendTraceSpecTo(t1[:0])

		Ne := 0

		ctx.Gin.Consume(func(v orca.Vtx, e orca.Edge) {
			if v.Label != 0 {
				X.vtx[v.Label-1] = VtxType(v.Color)
			} else {
				et := EdgeType(e.Color)
				X.edges[Ne] = et.FormEdge(VtxID(e.Va), VtxID(e.Vb))
				Ne++
			}
		})

		// ////
		//         X.traces = X.traces[:0]
		//         T2 := X.Traces(0).AppendTraceSpecTo(t2[:0])
		//         if !bytes.Equal(T1, T2) {
		//             panic("traces not equal after canonize")
		//         }

	}

	return nil
}
*/

func (stream *GraphStream) PermuteVtxSigns() *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		for Xsrc := range stream.Outlet {
			Xsrc.PermuteVtxSigns(func(Xperm *Graph) bool {
				X := NewGraph(Xperm)
				next.Outlet <- X
				return true
			})
			Xsrc.Reclaim()
		}
		next.Close()
	}()

	return next
}

func (stream *GraphStream) PermuteEdgeSigns() *GraphStream {
	next := &GraphStream{
		Outlet: make(chan *Graph, 1),
	}

	go func() {
		for Xsrc := range stream.Outlet {
			Xsrc.PermuteEdgeSigns(func(Xperm *Graph) bool {
				X := NewGraph(Xperm)
				next.Outlet <- X
				return true
			})
			Xsrc.Reclaim()
		}
		next.Close()
	}()

	return next
}
