package walker

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/art-media-platform/amp.SDK/stdlib/symbol"
	"github.com/art-media-platform/amp.SDK/stdlib/symbol/memory_table"
	"github.com/fine-structures/fine.SDK/go2x3"
	"github.com/fine-structures/fine.SDK/lib2x3/graph"
	//"github.com/fine-structures/fine.SDK/lib2x3/catalog"
)

func enumPureParticles(opts EnumOpts) (*go2x3.GraphStream, error) {
	//ctx := go2x3.NewCatalogContext()
	tableOpts := memory_table.DefaultOpts()
	emitted, err := tableOpts.CreateTable()
	if err != nil {
		return nil, err
	}
	gw := &graphWalker{
		opts:          opts,
		walkingVertex: 1,
		emitted:       emitted,
		EnumStream: &go2x3.GraphStream{
			Outlet: make(chan go2x3.State, 1),
		},
	}

	// Enqueue a single vertex particle
	gw.tryEmitFork(nil, GrowOp{
		OpCode: OpCode_Sprout,
		Count:  +1,
	})

	go func() {
		gw.emitSubParticles()
	}()

	return gw.EnumStream, nil
}

type graphWalker struct {
	EnumStream *go2x3.GraphStream
	forkCount  atomic.Uint64
	opts       EnumOpts
	emitted    symbol.Table

	walkingVertex int        // graph vtx size currently being emitted
	walkingQueue  GraphQueue // queue to process for current vtx size
	deferredQueue GraphQueue // queue to process for currentVtx + 1
}

var graphPool = sync.Pool{
	New: func() any {
		return &Construction{
			Vtx:    make([]graph.Vertex, 0, 32),
			Ops:    make([]GrowOp, 0, 64),
			traces: make([]int64, 0, 12),
		}
	},
}

type Construction struct {
	ParentID uint64         // instance ID
	ForkID   uint64         // instance ID
	Ops      []GrowOp       // build steps that yields State
	Vtx      []graph.Vertex // active vertex state
	Next     *Construction  // forward linked list
	traces   []int64        // traces storage
}

func (X *Construction) VertexCount() int {
	return len(X.Vtx)
}

func (X *Construction) Canonize(normalize bool) error {
	return nil
}

func (X *Construction) MarshalOut(out []byte, opts go2x3.MarshalOpts) ([]byte, error) {
	// TODO:
	panic("not implemented")
}

func (X *Construction) WriteCSV(out io.Writer, opts go2x3.PrintOpts) error {
	fmt.Fprintf(out, "p=%d,v=%d,", X.ParticleCount(), X.VertexCount())
	var buf [128]byte
	exprStr := X.marshalAsExpr(buf[:0], 1, true)
	exprStr = append(exprStr, ',')
	out.Write(exprStr)

	{
		for _, op := range X.Ops {
			rune := rune('?')
			switch {
			case op.OpCode == OpCode_Sprout && op.Count > 0:
				rune = '🌱'
			case op.OpCode == OpCode_Sprout && op.Count < 0:
				rune = '🌷'
			case op.OpCode == OpCode_AddEdge && op.Count > 0:
				rune = '🔵'
			case op.OpCode == OpCode_AddEdge && op.Count < 0:
				rune = '🟣'
			}
			fmt.Fprintf(out, "%02d%c", op.FromOrdinal(), rune)
		}
		fmt.Fprint(out, ",")
	}

	if opts.NumTraces != 0 {
		X.WriteTracesAsCSV(out, opts.NumTraces)
	}
	return nil
}

func (X *Construction) marshalAsExpr(out []byte, vtxID graph.VtxID, asAscii bool) []byte {
	out = append(out, '(')

	vtx := &X.Vtx[vtxID-1]
	for i, ei := range vtx.Edges {
		if ei.Path < 0 { // omit backward edges
			continue
		}

		asPos := byte('o')
		asNeg := byte('@')
		isAddEdge := false
		if ei.To != 0 {
			for j := 0; j < i; j++ {
				ej := vtx.Edges[j]
				if ej.Path > 0 && ej.To == ei.To {
					isAddEdge = true
					asPos = '+'
					asNeg = '-'
					break
				}
			}
		}

		if ei.To == 0 || isAddEdge {
			glyph := byte('?')
			if ei.Sign > 0 {
				glyph = asPos
			} else if ei.Sign < 0 {
				glyph = asNeg
			}
			out = append(out, glyph)
		} else {
			out = X.marshalAsExpr(out, ei.To, asAscii)
		}
	}
	return append(out, ')')
}

func (X *Construction) GraphInfo() go2x3.GraphInfo {
	return go2x3.GraphInfo{
		NumParticles: byte(X.ParticleCount()),
		NumVertex:    byte(X.VertexCount()),
	}
}

// Returns the number of particles (partitions) in this graph
func (X *Construction) ParticleCount() int64 {

	// We find number of total partitions.  Start by assuming each vertex its own partition.
	// Each time we connect two vertices with an edge, propagate their connectedness.
	var vtxBuf [go2x3.MaxVtxID]graph.VtxID
	Nv := graph.VtxID(len(X.Vtx))
	vtx := vtxBuf[:Nv]
	for i := graph.VtxID(0); i < Nv; i++ {
		vtx[i] = i + 1
	}
	// for _, edge := range X.Edges() {  FIX ME
	// 	va, vb := edge.VtxIdx()
	// 	v_lo := vtx[va]
	// 	v_hi := vtx[vb]
	// 	if v_lo == v_hi {
	// 		continue
	// 	}
	// 	if v_lo > v_hi {
	// 		v_lo, v_hi = v_hi, v_lo
	// 	}
	// 	for i, vi := range vtx {
	// 		if vi == v_hi {
	// 			vtx[i] = v_lo
	// 		}
	// 	}
	// }

	// The number of unique values in the vtx list is the number of partitions
	count := int64(0)
	if Nv > 0 {
		count++
	}
	for _, vi := range vtx {
		newPart := true
		for j := int64(0); j < count; j++ {
			if vtx[j] == vi {
				newPart = false
			}
		}
		if newPart {
			vtx[count] = vi
			count++
		}
	}

	return count

}

func (X *Construction) PermuteVtxSigns(dst *go2x3.GraphStream) {
	panic("legacy: will not implement")
}

// PermuteEdgeSigns emits a Graph for every possible edge sign permutation of the given Graph.
//
// The callback handler should not make any changes to Xperm (with the exception of calling Traces())
func (X *Construction) PermuteEdgeSigns(dst *go2x3.GraphStream) {

	dst.Outlet <- X.MakeCopy() // TODO

	/*
		// If there's no edges to permute over, export only the given graph (which is just 0 or more single vertex particles).
		// Note that X.edgeCount is vertex pair count, so 2 or 3 edges of matching type will only show up as *one* element.
		Ne := X.edgeCount
		if Ne == 0 {
			dst.Outlet <- X.MakeCopy()
			return
		}

		Xi := NewGraph(X)
		defer Xi.Reclaim()

		// Build the permutation we will traverse
		permCount := int64(1)
		var span [MaxEdges][4]EdgeID
		for ei, edgeID := range Xi.Edges() {
			edgePerm := edgeID.EdgePerm()
			span[ei] = edgePerm.Edges
			permCount *= int64(edgePerm.Num)
			Xi.edges[ei] = span[ei][0]
		}

		for {
			dst.Outlet <- Xi.MakeCopy()
			permCount--

			// "Increment" to the next permutation
			carry := true
			for ei := 0; ei < Ne && carry; ei++ {
				e := Xi.edges[ei]

				switch e {
				case span[ei][0]:
					e = span[ei][1]
				case span[ei][1]:
					e = span[ei][2]
				case span[ei][2]:
					e = span[ei][3]
				default:
					e = 0
				}

				// Is there a carry?
				if e == 0 {
					e = span[ei][0]
				} else {
					carry = false
				}

				// Write the edge change
				Xi.edges[ei] = e
			}

			Xi.onGraphChanged()

			if carry {
				if permCount != 0 {
					panic("calculated number of VtxType permutations did not equal number of enumerations")
				}
				break
			}
		}
	*/
}

func (X *Construction) Traces(numTraces int) go2x3.Traces {
	Nv := X.VertexCount()
	Nt := numTraces
	if Nt <= 0 {
		Nt = Nv
	}

	if cap(X.traces) < Nt {
		X.traces = make([]int64, (Nt+3)&^3)
	}
	TX := X.traces[:Nt]

	// init scrap
	NvNv := Nv * Nv
	scrap := make([]int64, NvNv*2)
	Ci0 := scrap[:NvNv]
	Ci1 := scrap[NvNv:]

	// init Ci0 with identity matrix
	for vi := range Nv {
		Ci0_vi := Ci0[Nv*vi : Nv*(vi+1)]
		for vj := range Nv {
			Ci0_vi[vj] = 0
		}
		Ci0_vi[vi] = 1 // or any k[vj]
	}

	for ti := range Nt {
		TX_ci := int64(0)

		for vi := range Nv {
			Ci0_vi := Ci0[Nv*vi : Nv*(vi+1)]
			Ci1_vi := Ci1[Nv*vi : Nv*(vi+1)]

			for j, vj := range X.Vtx {
				totalFlow := int64(0)

				for _, vj_e := range vj.Edges {

					// pull flow from previous state
					v_src := vj_e.To // outward edge
					if v_src == 0 {
						v_src = vj.ID // inward edge
					}
					edgeFlow := Ci0_vi[v_src-1]
					if vj_e.Sign < 0 {
						edgeFlow = -edgeFlow
					}
					totalFlow += edgeFlow
				}

				Ci1_vi[j] = totalFlow // write into next state
			}

			TX_ci += Ci1_vi[vi] // accumulate cycle counts of length i
		}

		TX[ti] = TX_ci

		// swap previous and next states toå advance
		Ci0, Ci1 = Ci1, Ci0
	}

	return TX
}

func (X *Construction) MakeCopy() go2x3.State {
	return NewState(X)
}

// func (X *Construction) WriteAsGraphExprStr(out io.Writer) {
// 	for _, vi := range X.Vtx {
// 		fmt.Fprintf(out, "%d:", vi.ID)
// 		for _, ej := range vi.Edges {
// 			if ej.To == 0 {
// 				continue
// 			}
// 			fmt.Fprintf(out, "%d", ej.To)
// 			if ej.Sign == Sign_Invert {
// 				out.Write([]byte{'-'})
// 			} else {
// 				out.Write([]byte{'+'})
// 			}
// 			out.Write([]byte{' '})
// 		}
// 		out.Write([]byte{'\n'})
// 	}
// }

func (X *Construction) WriteTracesAsCSV(out io.Writer, numTraces int) {
	TX := X.Traces(numTraces)

	var buf [24]byte

	for _, TXi := range TX {
		out.Write(graph.PrintInt(buf[:], TXi))
		out.Write([]byte{','})
	}
}

// Recycles this State instance into a pool for reuse.
// Caller asserts that no more references to this instance will persist.
func (X *Construction) Reclaim() {
	for X != nil {
		next := X.Next
		X.Next = nil
		graphPool.Put(X)
		X = next
	}
}

// Returns true if the given graph is unique
func (gw *graphWalker) isUnique(X *Construction) bool {
	TX := X.Traces(0)

	var scrap [128]byte
	sym := TX.AppendTracesLSM(scrap[:0])
	_, newlyIssued := gw.emitted.GetSymbolID([]byte(sym), true)
	return newlyIssued
}

/*
func (X *Construction) traces(tmp *graph.VtxGraphVM) go2x3.Traces {
	tmp.ResetGraph()
	for _, vi := range X.Vtx {
		for _, ei := range vi.Edges {
			vj := ei
			if ei == 0 {
				vj = vi.ID
			}
			if vi.ID <= vj { // only add each edge once
				tmp.AddEdge(0, 1, uint32(vi.ID), uint32(vj))
			}
		}
	}
	tmp.Validate()
	return tmp.Traces(0)
}
*/

/*
func (X *Construction) recountSiblings(vi VtxID) {
	slots := &X.Vtx[vi-1].Edges

	// pass 1: (re)count siblings
	for i, si := range slots {
		siblings := int8(0)
		if si.OtherID != 0 {
			for _, sj := range slots {
				if si.OtherID == sj.OtherID {
					siblings += 1
				}
			}
		}
		slots[i].Siblings = siblings
	}
}
*/

func NewState(Xsrc *Construction) *Construction {
	X := graphPool.Get().(*Construction)
	X.Next = nil
	X.traces = X.traces[:0]
	if Xsrc != nil {
		X.ForkID = Xsrc.ForkID
		X.ParentID = Xsrc.ForkID
		X.Vtx = append(X.Vtx[:0], Xsrc.Vtx...)
		X.Ops = append(X.Ops[:0], Xsrc.Ops...)
	} else {
		X.ForkID = 0
		X.ParentID = 0
		X.Vtx = X.Vtx[:0]
		X.Ops = X.Ops[:0]
	}
	return X
}

func (X *Construction) NegateEdge(vi graph.VtxID, vi_slot int32) {
	if vi <= 0 || vi > graph.VtxID(len(X.Vtx)) || vi_slot > graph.EdgesPerVertex {
		panic("NegateEdge: invalid edge")
	}

}

// findEdge returns the vertex and slot of the edge that connects to the given vertex and slot
func (X *Construction) findEdge(vi graph.VtxID, vi_slot byte) (vj graph.VtxID, vj_slot_edge, vj_slot_free byte) {
	if vi <= 0 || int(vi) > len(X.Vtx) || vi_slot == 0 || vi_slot > graph.EdgesPerVertex {
		return // invalid input
	}
	vj = X.Vtx[vi-1].Edges[vi_slot-1].To
	if vj == 0 {
		return // no edge found
	}
	for j, eb := range X.Vtx[vj-1].Edges {
		if eb.To == vi { // found matching edge
			if vj_slot_edge == 0 {
				vj_slot_edge = byte(j + 1)
			}
		} else if eb.To == 0 { // found open slot
			if vj_slot_free == 0 {
				vj_slot_free = byte(j + 1)
			}
		}
	}
	return
}

func (X *Construction) findOpenSlot(vi graph.VtxID) (vi_slot byte) {
	for i, ej := range X.Vtx[vi-1].Edges {
		if ej.To == 0 {
			return byte(i + 1)
		}
	}
	return 0
}

func (X *Construction) applyOp(op GrowOp) bool {

	// base case: sprout a new vertex
	if len(X.Vtx) == 0 {
		X.addNewVertex()
		return true
	}

	vtxA := op.FromVtx
	vtxB := graph.VtxID(0)
	slotA := byte(op.FromSlot)
	slotB := byte(0)
	if vtxA <= 0 || slotA == 0 || slotA > graph.EdgesPerVertex {
		return false
	}

	switch op.OpCode {
	case OpCode_AddEdge:
		vtxB, _, slotB = X.findEdge(vtxA, slotA)
		slotA = X.findOpenSlot(vtxA)
	case OpCode_Sprout:
		newVtxID := X.addNewVertex()
		vtxB, slotB, _ = X.findEdge(vtxA, slotA)

		newVtx := &X.Vtx[newVtxID-1]
		if vtxB > 0 {
			newVtx.Edges[1] = graph.Edge{ // re-attach vtxB to new vtx
				To:   vtxB,
				Sign: +1,
				Path: +1,
			}
			X.Vtx[vtxB-1].Edges[slotB-1] = graph.Edge{ // re-attach vtxB to new vtx
				To:   newVtxID,
				Sign: +1,
				Path: -1,
			}
		}

		vtxB = newVtxID
		slotB = 1
	}

	if vtxA == 0 || slotA == 0 || vtxB == 0 || slotB == 0 {
		return false
	}

	X.Vtx[vtxA-1].Edges[slotA-1] = graph.Edge{
		To:   vtxB,
		Sign: +1,
		Path: +1,
	}
	X.Vtx[vtxB-1].Edges[slotB-1] = graph.Edge{
		To:   vtxA,
		Sign: +1,
		Path: -1,
	}

	return true
}

func (gw *graphWalker) tryEmitFork(X0 *Construction, op GrowOp) {
	X := NewState(X0)
	ok := X.applyOp(op)

	Nv := X.VertexCount()
	if !ok || Nv > gw.opts.VertexMax || !gw.isUnique(X) {
		X.Reclaim()
		return
	}
	X.ForkID = gw.forkCount.Add(1)
	X.Ops = append(X.Ops, op)

	if Nv <= gw.walkingVertex {
		gw.walkingQueue.Enqueue(X)
	} else {
		gw.deferredQueue.Enqueue(X)
	}
}

type GraphQueue struct {
	Head  *Construction
	Tail  *Construction
	Count int
}

func (queue *GraphQueue) Enqueue(X *Construction) {
	X.Next = nil
	if queue.Tail != nil {
		queue.Tail.Next = X
	}
	queue.Tail = X
	if queue.Head == nil {
		queue.Head = X
	}
	queue.Count++
}

func (queue *GraphQueue) Dequeue() *Construction {
	X := queue.Head
	if X == nil {
		return nil
	}
	queue.Head = X.Next
	X.Next = nil
	if queue.Tail == X {
		queue.Tail = nil
	}
	queue.Count--
	return X
}

/*
func (X *Construction) popStep() bool {

	// pop the most recent grow step
	N := len(X.Ops)
	if N == 0 {
		return false
	}
	N--
	undo := X.Ops[N]
	X.Ops = X.Ops[:N]

	// If the op added a vertex, remove it
	switch undo.OpCode {
	case graph.OpCode_EdgeSplit, graph.OpCode_Sprout:
		Nv := len(X.Vtx)
		X.Vtx = X.Vtx[:Nv-1]
	}

	// revert the affected slots
	slotA := &X.Vtx[undo.VtxA-1].Edges[undo.SlotA-1]
	slotB := &X.Vtx[undo.VtxB-1].Edges[undo.SlotB-1]

	switch undo.OpCode {
	case graph.OpCode_EdgeDuplicate, graph.OpCode_Sprout:
		*slotA = 0
		*slotB = 0
	case graph.OpCode_EdgeSplit:
		*slotA = undo.VtxB
		*slotB = undo.VtxA
	}


		// switch undo.OpCode {
		// case OpCode_EdgeSplit, OpCode_EdgeDuplicate:
		// 	X.recountSiblings(undo.VtxA)
		// 	X.recountSiblings(undo.VtxB)
		// case OpCode_Sprout:
		// 	slotA.Siblings = 0 // no need to recount siblings
		// }

	return true
}
*/

func (gw *graphWalker) doubleEdges(X *Construction) {
	op := GrowOp{
		OpCode: OpCode_AddEdge,
		Count:  1,
	}

	for i := range X.Vtx {
		va := &X.Vtx[i]

		op.FromVtx = 0
		freeSlot := int8(0)

		for j, ej := range va.Edges {

			if ej.To == 0 { // look for free slot on local vertex
				if freeSlot == 0 {
					freeSlot = int8(j + 1)
				}
			} else if ej.Path < 0 { // skip reverse direction (equivalent)
				continue
			} else if op.FromVtx == 0 {
				op.FromVtx = va.ID
				op.FromSlot = uint8(j + 1)
			}
		}

		// this vertex has no free slots or no edges to duplicate
		if op.FromVtx == 0 || freeSlot == 0 {
			continue
		}

		gw.tryEmitFork(X, op)
	}
}

func (X *Construction) addNewVertex() (newVtxID graph.VtxID) {
	newVtxID = graph.VtxID(len(X.Vtx) + 1)
	v := graph.Vertex{
		ID: newVtxID,
	}
	for ei := range v.Edges {
		v.Edges[ei].Sign = +1
	}
	X.Vtx = append(X.Vtx, v)
	return newVtxID
}

func (gw *graphWalker) sproutEdges(X *Construction) {
	if X.VertexCount() >= gw.opts.VertexMax {
		return
	}
	vtx := X.Vtx
	for i := range vtx {
		va := &vtx[i]

		for j, ej := range va.Edges {
			if ej.Path < 0 { // no need to split "backward" edges
				continue
			}

			gw.tryEmitFork(X, GrowOp{
				OpCode:   OpCode_Sprout,
				Count:    1,
				FromVtx:  va.ID,
				FromSlot: uint8(j + 1),
			})
		}
	}
}

func (gw *graphWalker) emitSubParticles() {
	var X *Construction
	for X = gw.dequeueNext(); X != nil; X = gw.dequeueNext() {

		// fork 1: iF we can duplicate an edge, then do so.
		gw.doubleEdges(X)

		// fork 2 -- "sprout" a new vertex from an edge slot
		gw.sproutEdges(X)

		// after emitting all possible forks, emit outward
		gw.EnumStream.Outlet <- X
	}

	gw.EnumStream.Close()
}

func (gw *graphWalker) dequeueNext() *Construction {
	if gw.walkingQueue.Count == 0 && gw.deferredQueue.Count > 0 {
		gw.walkingVertex++
		gw.deferredQueue, gw.walkingQueue = gw.walkingQueue, gw.deferredQueue
	}
	X := gw.walkingQueue.Dequeue()
	return X
}
