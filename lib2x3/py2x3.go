package lib2x3

// Copyright 2018 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/go-python/gpython/py"
)

var (
	LIB_VERSION = "v1.2023.1"
)

var (
	PyGraphType       = py.NewType("Graph", "an opaque 2x3 object containing zero or more particles")
	PyGraphStreamType = py.NewType("GraphStream", "lib2x3.GraphStream")
	PyCatalogType     = py.NewType("Catalog", "lib2x3.Catalog")
	PyWorkspaceType   = py.NewType("Workspace", "collets active session resources and catalogs")
)

// Arg 1 (int): Nv_start
// Arg 2 (int): Nv_end
func ph_EnumPureParticles(module py.Object, args py.Tuple) (py.Object, error) {
	var v_min, v_max py.Object
	err := py.ParseTuple(args, "ii", &v_min, &v_max)
	if err != nil {
		return nil, err
	}

	n0 := int(v_min.(py.Int))
	n1 := int(v_max.(py.Int))
	stream := EnumPureParticles(n0, n1, "BackConnect.1")
	return py.Object(stream), nil
}

func getGraphFromGraphObj(obj py.Object) (X *Graph, err error) {
	if obj.Type().Name != "Graph" {
		err = py.ExceptionNewf(py.TypeError, "expected Graph object (got %v)", obj.Type().Name)
		return
	}
	var attr py.Object
	attr, err = py.GetAttrString(obj, "_graph")
	if err != nil {
		return
	}
	X = attr.(*Graph)
	return
}

func (X *Graph) Type() *py.Type {
	return PyGraphType
}

func (X *Graph) M__str__() (py.Object, error) {
	writer := strings.Builder{}
	X.WriteAsString(&writer, DefaultPrintOpts)
	return py.String(writer.String()), nil
}

func (X *Graph) M__repr__() (py.Object, error) {
	return X.M__str__()
}

func ph_NewGraph(module py.Object, args py.Tuple) (py.Object, error) {
	X := NewGraph(nil)
	return py.Object(X), nil
}

func ph_Graph_NumVerts(self py.Object, args py.Tuple) (py.Object, error) {
	X := self.(*Graph)
	return py.Object(py.Int(X.NumVerts())), nil
}

func ph_Graph_NumParts(self py.Object, args py.Tuple) (py.Object, error) {
	X := self.(*Graph)
	return py.Object(py.Int(X.NumParticles())), nil
}

func ph_Graph_Traces(self py.Object, args py.Tuple) (py.Object, error) {
	X := self.(*Graph)
	numTraces := 0
	if len(args) > 0 {
		numTraces = int(args[0].(py.Int))
	}

	TX := X.Traces(numTraces)

	N := len(TX)
	traces := make(py.Tuple, N)
	for i, tr := range TX {
		traces[i] = py.Int(tr)
	}

	return py.Object(traces), nil
}

func ph_Graph_Concat(self py.Object, args py.Tuple) (py.Object, error) {
	X := self.(*Graph)
	srcGraphs := args[0].(py.Tuple)
	var Xi Graph

	for i, arg := range srcGraphs {

		if initStr, isStr := arg.(py.String); isStr {
			err := Xi.InitFromString(string(initStr))
			if err != nil {
				return nil, py.ExceptionNewf(py.TypeError, "error reading part %d: %v", i, err)
			}
			X.Concatenate(&Xi)

		} else {
			Xsrc, err := getGraphFromGraphObj(arg)
			if err != nil {
				return nil, err
			}
			X.Concatenate(Xsrc)
		}
	}

	return py.Object(X), nil
}

func ph_Graph_Stream(self py.Object, args py.Tuple) (py.Object, error) {
	X := self.(*Graph)
	next := StreamGraph(X)
	return py.Object(next), nil
}

const (
	READ_ONLY     = 0x01
	PRIME_CATALOG = 0x04

	kWorkspaceAttr = "_Workspace"
)

type Workspace struct {
	CatalogCtx CatalogContext
	//	Stdout     *py.File
}

func (ws *Workspace) Close() {
	ws.CatalogCtx.Close()
	<-ws.CatalogCtx.Done()
}

func (ws *Workspace) Type() *py.Type {
	return PyWorkspaceType
}

func ph_GetWorkspace(module py.Object, args py.Tuple) (py.Object, error) {
	wsObj, _ := py.GetAttrString(module, kWorkspaceAttr)
	if wsObj == nil {
		ws := &Workspace{
			CatalogCtx: NewCatalogContext(),
			//Stdout:     module.(*py.Module).Context.Store().MustGetModule("sys").Globals["stdout"].(*py.File),
		}
		wsObj = ws
		py.SetAttrString(module, kWorkspaceAttr, wsObj)
	}
	return wsObj, nil
}

func ph_Workspace_CatalogExists(self py.Object, args py.Tuple) (py.Object, error) {
	_ = self.(*Workspace)

	var pathname string
	err := py.LoadTuple(args, []interface{}{&pathname})
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(pathname)
	if os.IsNotExist(err) {
		return py.False, nil
	}
	return py.True, nil
}

func ph_Workspace_OpenCatalog(self py.Object, args py.Tuple) (py.Object, error) {
	ws := self.(*Workspace)

	var pathname string
	var flags, minTraceCount int32
	err := py.LoadTuple(args, []interface{}{&pathname, &flags, &minTraceCount})
	if err != nil {
		return nil, err
	}

	opts := CatalogOpts{
		ReadOnly:   (flags & READ_ONLY) != 0,
		DbPathName: pathname,
		TraceCount: minTraceCount,
	}
	if (flags & PRIME_CATALOG) != 0 {
		opts.NeedPrimes = true
	}

	cat, err := ws.CatalogCtx.OpenCatalog(opts)
	if err != nil {
		return nil, py.ExceptionNewf(py.RuntimeError, "%v", err)
	}

	return py.Object(cat), nil
}

func ph_Catalog_Close(self py.Object, args py.Tuple) (py.Object, error) {
	cat := self.(Catalog)
	if cat != nil {
		cat.Close()
	}
	return py.None, nil
}

func ph_Catalog_Select(self py.Object, args py.Tuple) (py.Object, error) {
	cat := self.(Catalog)
	sel := DefaultGraphSelector
	if len(args) > 0 {
		err := getGraphSelector(args[0], &sel)
		if err != nil {
			return nil, err
		}
	}

	next := SelectFromCatalog(cat, sel)
	return py.Object(next), nil
}

func ph_Catalog_NumTraces(self py.Object, args py.Tuple) (py.Object, error) {
	cat := self.(Catalog)

	Nv, err := py.GetInt(args[0])
	if err != nil {
		return nil, err
	}

	numTraces := cat.NumTraces(byte(Nv))
	return py.Int(numTraces), nil
}

func ph_Catalog_NumPrimes(self py.Object, args py.Tuple) (py.Object, error) {
	cat := self.(Catalog)

	Nv, err := py.GetInt(args[0])
	if err != nil {
		return nil, err
	}

	numPrimes := cat.NumPrimes(byte(Nv))
	return py.Int(numPrimes), nil
}

func (stream *GraphStream) Type() *py.Type {
	return PyGraphStreamType
}

func ph_GraphStream_Go(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	count := stream.PullAll()
	return py.Int(count), nil
}

type echoToWriter struct {
	stdout *os.File
	to     io.WriteCloser
}

func (echo *echoToWriter) Write(buf []byte) (int, error) {
	var (
		n   int
		err error
	)
	if echo.to == nil {
		n, err = echo.stdout.Write(buf)
	} else {
		n, err = echo.to.Write(buf)
	}
	return n, err
}

func (echo *echoToWriter) Close() error {
	if echo.to != nil {
		return echo.to.Close()
	}
	return nil
}

var gOutCount = int32(0)

// See lib/py2x3.py Print() docs
func ph_GraphStream_Print(self py.Object, args py.Tuple, kwargs py.StringDict) (py.Object, error) {
	stream := self.(*GraphStream)
	var pathname string

	opts := DefaultPrintOpts

	py.LoadTuple(args, []interface{}{&opts.Label})
	if opts.Label == "" {
		py.LoadAttr(kwargs, "label", &opts.Label)
	}

	// TODO: move this to the Workspace obj so output counter is within the workspace (vs global)
	atomic.AddInt32(&gOutCount, 1)
	if opts.Label == "" {
		opts.Label = fmt.Sprintf("out[%d]", gOutCount)
	}

	py.LoadAttr(kwargs, "traces", &opts.NumTraces)
	py.LoadAttr(kwargs, "cycles", &opts.CycleSpec)
	py.LoadAttr(kwargs, "codes", &opts.Tricodes)
	py.LoadAttr(kwargs, "matrix", &opts.Matrix)
	py.LoadAttr(kwargs, "graph", &opts.Graph)
	py.LoadAttr(kwargs, "file", &pathname)

	// See TODO on also allowing output object instead of filename
	writer := &echoToWriter{
		stdout: os.Stdout,
	}
	if len(pathname) > 0 {
		os.MkdirAll(filepath.Dir(pathname), 0700)

		file, err := os.OpenFile(string(pathname), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, py.ExceptionNewf(py.FileNotFoundError, "%v", err)
		}
		writer.to = file
	}

	next := stream.Print(writer, opts)
	return py.Object(next), nil
}

/*
func ph_NewGraphStream(module py.Object) (py.Object, error) {
	//
	// TODO: add mechanism to close all open GraphStreams when a script finishes
	//
	{
	}
	stream := NewGraphStream()
	return py.Object(stream), nil
}

func ph_GraphStream_PushGraph(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	X := args[0].(*Graph)
	// attr, err := py.GetAttrString(args[0], "_graph")
	// if err != nil {
	// 	return nil, err
	// }
	// X := attr.(*Graph)
	if X == nil {
		return nil, py.ExceptionNewf(py.TypeError, "%v", errors.New("expected Graph type object"))
	}
	stream.PushGraph(X)
	return py.None, nil
}

func ph_GraphStream_PullGraph(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	X := stream.PullGraph()
	if X == nil {
		return py.None, nil
	}
	return py.Object(X), nil
	// py.SetAttrString(args[0], "_graph", py.Object(X))
	// return py.True, nil
}
*/

func ph_GraphStream_AddTo(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	attr, err := py.GetAttrString(args[0], "_cat")
	if err != nil {
		return nil, err
	}
	cat := attr.(Catalog)
	if cat.IsReadOnly() {
		return nil, py.ExceptionNewf(py.PermissionError, "%v", errors.New("Catalog is in read-only mode"))
	}

	next := stream.AddTo(cat, AddGraphOpts{})
	return py.Object(next), nil
}

func ph_GraphStream_DropDupes(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)

	// Create a memory resident catalog that get auto-closed when the stream closes
	cat := NewDropDupes(DropDupeOpts{})
	opts := AddGraphOpts{
		AutoCloseCatalog: true,
	}

	next := stream.AddTo(cat, opts)
	return py.Object(next), nil
}

func ph_GraphStream_Canonize(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	normalize := false
	err := py.LoadTuple(args, []interface{}{&normalize})
	if err != nil {
		return nil, err
	}
	next := stream.Canonize(normalize)
	return py.Object(next), nil
}

func ph_GraphStream_Select(self py.Object, args py.Tuple) (py.Object, error) {
	sel := DefaultGraphSelector
	err := getGraphSelector(args[0], &sel)
	if err != nil {
		return nil, err
	}
	stream := self.(*GraphStream)
	next := stream.SelectFromStream(sel)
	return py.Object(next), nil
}

func ph_GraphStream_PermuteVtxSigns(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	next := stream.PermuteVtxSigns()
	return py.Object(next), nil
}

func ph_GraphStream_PermuteEdgeSigns(self py.Object, args py.Tuple) (py.Object, error) {
	stream := self.(*GraphStream)
	next := stream.PermuteEdgeSigns()
	return py.Object(next), nil
}

func init() {

	/////////////////////////////////
	// Graph
	{
		PyGraphType.Dict["Traces"] = py.MustNewMethod("Traces", ph_Graph_Traces, 0, "exports this Graph's Traces as a bytes object")
		PyGraphType.Dict["NumVerts"] = py.MustNewMethod("NumVerts", ph_Graph_NumVerts, 0, "")
		PyGraphType.Dict["NumParts"] = py.MustNewMethod("NumParts", ph_Graph_NumParts, 0, "")
		PyGraphType.Dict["Concat"] = py.MustNewMethod("Concat", ph_Graph_Concat, 0, "")
		PyGraphType.Dict["Stream"] = py.MustNewMethod("Stream", ph_Graph_Stream, 0, "")
	}

	/////////////////////////////////
	// Catalog
	{
		PyCatalogType.Dict["Select"] = py.MustNewMethod("Select", ph_Catalog_Select, 0, "")
		PyCatalogType.Dict["NumTraces"] = py.MustNewMethod("NumTraces", ph_Catalog_NumTraces, 0, "")
		PyCatalogType.Dict["NumPrimes"] = py.MustNewMethod("NumPrimes", ph_Catalog_NumPrimes, 0, "")
		PyCatalogType.Dict["Close"] = py.MustNewMethod("Close", ph_Catalog_Close, 0, "")
	}

	/////////////////////////////////
	// Workspace
	{
		PyWorkspaceType.Dict["OpenCatalog"] = py.MustNewMethod("OpenCatalog", ph_Workspace_OpenCatalog, 0, "")
		PyWorkspaceType.Dict["CatalogExists"] = py.MustNewMethod("CatalogExists", ph_Workspace_CatalogExists, 0, "")
	}

	/////////////////////////////////
	// GraphStream
	{
		PyGraphStreamType.Dict["Go"] = py.MustNewMethod("Go", ph_GraphStream_Go, 0, "counts the number of graphs output from the GraphStream")
		PyGraphStreamType.Dict["Print"] = py.MustNewMethod("Print", ph_GraphStream_Print, 0, "prints each graph from the GraphStream")
		// PyGraphStreamType.Dict["PullGraph"] = py.MustNewMethod("PullGraph", ph_GraphStream_PullGraph, 0, "")
		// PyGraphStreamType.Dict["PushGraph"] = py.MustNewMethod("PushGraph", ph_GraphStream_PushGraph, 0, "")
		PyGraphStreamType.Dict["AddTo"] = py.MustNewMethod("AddTo", ph_GraphStream_AddTo, 0, "")
		PyGraphStreamType.Dict["Canonize"] = py.MustNewMethod("Canonize", ph_GraphStream_Canonize, 0, "")
		PyGraphStreamType.Dict["DropDupes"] = py.MustNewMethod("DropDupes", ph_GraphStream_DropDupes, 0, "")
		PyGraphStreamType.Dict["Select"] = py.MustNewMethod("Select", ph_GraphStream_Select, 0, "")
		PyGraphStreamType.Dict["AllVtxSigns"] = py.MustNewMethod("AllVtxSigns", ph_GraphStream_PermuteVtxSigns, 0, "")
		PyGraphStreamType.Dict["AllEdgeSigns"] = py.MustNewMethod("AllEdgeSigns", ph_GraphStream_PermuteEdgeSigns, 0, "")

	}

	{
		methods := []*py.Method{
			py.MustNewMethod("NewGraph", ph_NewGraph, 0, ""),
			//py.MustNewMethod("GraphStream", ph_NewGraphStream, 0, ""),
			py.MustNewMethod("EnumPureParticles", ph_EnumPureParticles, 0, ""),
			py.MustNewMethod("GetWorkspace", ph_GetWorkspace, 0, ""),
		}

		// stdin, stdout, stderr := &py.String{os.Stdin, py.FileRead},
		// 	&py.File{os.Stdout, py.FileWrite},
		// 	&py.File{os.Stderr, py.FileWrite}
		globals := py.StringDict{
			"LIB_VERSION": py.String(LIB_VERSION),
			"PY_VERSION":  py.String("v3.4.0"),
			"MAX_VTX":     py.Int(MaxVtxID),
		}

		py.RegisterModule(&py.ModuleImpl{
			Info: py.ModuleInfo{
				Name: "_py2x3",
				Doc:  "2x3 Particle Theory gpython module",
			},
			Methods: methods,
			Globals: globals,
			OnContextClosed: func(m *py.Module) {
				wsObj, _ := py.GetAttrString(m, kWorkspaceAttr)
				if wsObj != nil {
					wsObj.(*Workspace).Close()
				}
			},
		})

	}
}

func intAttr(obj py.Object, key string, min, max int64) int64 {
	attr, err := py.GetAttrString(obj, key)
	if err != nil {
		panic(err)
	}
	val, _ := py.GetInt(attr)
	intVal := int64(val)
	if intVal < min {
		intVal = min
	}
	if intVal > max {
		intVal = max
	}
	return intVal
}

func byteAttr(obj py.Object, attr string) byte {
	return byte(intAttr(obj, attr, 0, 255))
}

func exportGraphInfo(graphInfo py.Object) GraphInfo {
	info := GraphInfo{
		NumParticles: byteAttr(graphInfo, "parts"),
		NumVerts:     byteAttr(graphInfo, "verts"),
		PosEdges:     byteAttr(graphInfo, "pos_edges"),
		NegEdges:     byteAttr(graphInfo, "neg_edges"),
		PosLoops:     byteAttr(graphInfo, "pos_loops"),
		NegLoops:     byteAttr(graphInfo, "neg_loops"),
	}
	return info
}

func getGraphSelector(graph_selector py.Object, sel *GraphSelector) error {

	info, err := py.GetAttrString(graph_selector, "min")
	if err != nil {
		return err
	}
	sel.Min = exportGraphInfo(info)

	info, err = py.GetAttrString(graph_selector, "max")
	if err != nil {
		return err
	}
	sel.Max = exportGraphInfo(info)

	if err = py.LoadAttr(graph_selector, "factor", &sel.Factor); err != nil {
		return err
	}

	if err = py.LoadAttr(graph_selector, "primes", &sel.PrimesOnly); err != nil {
		return err
	}

	if err = py.LoadAttr(graph_selector, "unique_traces", &sel.UniqueTraces); err != nil {
		return err
	}

	if sel.Factor && (sel.PrimesOnly || sel.UniqueTraces) {
		return py.ExceptionNewf(py.ValueError, "%v", errors.New("'factor' mode can't be used with 'primes' or 'unique_traces'"))
	}

	tracesObj, err := py.GetAttrString(graph_selector, "traces")
	if err != nil {
		return err
	}
	getGraphFromGraphObj(tracesObj)

	switch tracesObj.(type) {
	// case py.Tuple, *py.List:
	// 	sel.Traces, err = py.LoadIntsFromList(tracesObj)
	// 	if err != nil {
	// 		return err
	// 	}
	case py.NoneType:
		sel.Traces = nil
	default:
		X, err := getGraphFromGraphObj(tracesObj)
		if err != nil {
			return err
		}
		sel.Min.NumVerts = X.NumVerts()
		sel.Max.NumVerts = X.NumVerts()
		sel.Traces = X
	}

	/*
		// case py.Tuple:
			numTraces, err = py.GetLen(traces)
			if err != nil {
				return err
			}
		}

		getter, ok := traces.(py.I__getitem__)
		if ok && numTraces > 0 {
			sel.Traces = make(Traces, numTraces)

			var intVal py.Int
			for i := py.Int(0); i < numTraces; i++ {
				item, err := getter.M__getitem__(i)
				if err != nil {
					return err
				}

				intVal, err = py.GetInt(item)
				if err != nil {
					return err
				}

				sel.Traces[i] = int64(intVal)
			}
	*/

	return nil
}
