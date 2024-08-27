package go2x3

import (
	"io"

	"github.com/art-media-platform/amp-sdk-go/stdlib/generics"
)

const (

	// VtxMax is the max possible value of a VtxID (a one-based index).
	MaxVtxID = 31

	// VtxIDBits is the number of bits dedicated for a VtxID.  It must be enough bits to represent MaxVtxID.
	VtxIDBits byte = 5

	// MaxEdgeEnds is the max number of possible edge connections for the largest graph possible.
	MaxEdges    = 3 * MaxVtxID / 2
	MaxEdgeEnds = 3 * MaxVtxID
)

type ExportOpts int32

const (
	ExportAsAscii ExportOpts = 1 << iota
	ExportGraphState
	ExportGraphDef
)

type AsciiDigiOrd byte

type Octal byte // 0..7

/*
type Encoding struct {
	byte     Octal
	Commands []tag.CanonicOctalDigit

}

*/

type System interface {
	AssignFromEncoding()

	// From the current graph.Encoding, constructs this graph.GraphState.
	Recompute() error

	Canonize() error
}

// func (gb *graphWalker) AssignFromEncoding(enc Encoding) {

// }

// var ordinalMap = [128]byte {
// 	0,
// 	1, 2, 3, 4, 5, 6, 7, 8,
// |


// func (gb *graphBuilder) EmitNextEdgeSweep(uptoShapeID uint64, tag.OctalDigit) {

// }

// ShapeID   uint64 // canonic enumeration step of a 2x3 tree instance "walk"
// VariantID uint64 // canonical "complete" natural 2x3 graph (0 corresponds to all positive edge variant)

/*
func (gb *graphBuilder) EnumerateStructureShapes(uptoShapeID uint64, ) {
	sockets := make([]VtxSocket, 3*MaxVtxID)


	{
		si := gb.addVertex()


		gb.PushState()
		gb.sweepPhase = 0
		for kind := 0; kind < 8; kind++ {

		}
		gb.sockets[vi].EdgeKind = 0
		gb.sockets[vi].ConnectsTo =
	}






}
// */

// func (gw *graphWalker) addVertex() byte {
// 	gw.vertexCount++
// 	vtxID := gw.vertexCount
// 	socket := VtxSocket{
// 		VertexID: vtxID,
// 	}
// 	for i := 0; i < 3; i++ {
// 		gw.sockets = append(gw.sockets, socket)
// 	}
// 	return vtxID
// }

// func (gw *graphWalker) Recompute() error {

// }

type GraphState interface {
	TracesProvider

	PermuteEdgeSigns(dst *GraphStream)

	Canonize(normalize bool) error

	WriteAsString(out io.Writer, opts PrintOpts)
	ExportStateEncoding(out []byte, opts ExportOpts) ([]byte, error)

	// Returns a new copy of this instance.
	MakeCopy() GraphState

	// Returns info about this graph
	GetInfo() GraphInfo

	// Recycles this GraphState instance into a pool for reuse.
	// Caller asserts that no more references to this instance will persist.
	Reclaim()
}

type TracesProvider interface {
	VertexCount() int
	Traces(numTraces int) Traces
}

// Traces is a sequence of a 2x3 graph "traces" values (successive powers of the graph adjacency matrix).
type Traces []int64

// TracesLSM is a LSM binary encoding / symbol of a Traces.
type TracesLSM []byte

// TracesID uniquely identifies a cycle trace series
type TracesID uint64

// OnGraphHit is a callback proc used to return Graph's meeting a set of selection criteria.
// Ownership of a Graph also travels through the channel.
type OnGraphHit chan<- GraphState

// CatalogContext is a container for open / active Catalog instances.
type CatalogContext interface {

	// Attaches the given Catalog to this context.
	AttachCatalog(cat Catalog)

	// Detaches the given Catalog from this context.
	DetachCatalog(cat Catalog)

	// Closes all open catalogs to be closed then closes.
	Close()

	// Signals when Close() completed and all open Catalogs have been closed
	Done() <-chan struct{}
}

// CatalogOpts specifies params for opening a lib2x3 Catalog
type CatalogOpts struct {
	DbPathName string // omit or for in-memory db
	ReadOnly   bool   // open in read-only mode
	TraceCount int32  // number of traces to preallocate
	NeedPrimes bool   // set if this catalog will be used for prime detection
}

type GraphAdder interface {

	// Tries to add the given graph encoding to this catalog.
	// If true is returned, X did not exist and was added.
	TryAddGraph(X GraphState) bool
}

// Catalog wraps a database of lib2x3 Graph encodings.
type Catalog interface {
	GraphAdder

	// Returns true if this catalog was opened for read-only access.
	IsReadOnly() bool

	// NumTraces returns the number of unique Traces in this catalog for a given vertex count.
	// NumTraces()[0] is always 0 and an out of bounds vertex count returns 0.
	NumTraces(forVtxCount byte) int64

	// NumPrimes returns the number of particle primes for a given vertex count (one-based indexing).
	// NumPrimes()[0] is always 0 and an out of bounds vertex count returns 0.
	NumPrimes(forVtxCount byte) int64

	// Select fires the given callback with each GraphEncoding that meets the selection criteria.
	Select(sel GraphSelector, onHit OnGraphHit)

	Close() error
}

type GraphInfo struct {
	NumParticles byte
	NumVertex    byte
	NegLoops     byte
	PosLoops     byte
	NegEdges     byte
	PosEdges     byte
}

// GraphSelector is an operator that either selects a given Graph or not.
type GraphSelector struct {
	Traces       TracesProvider // Implies a Traces to match with or factor
	Factor       bool           // Perform factorization of sel.Traces
	UniqueTraces bool           // Only select the first Graph for each unique traces
	PrimesOnly   bool           // Only select prime traces
	Min          GraphInfo      // lower select bounds
	Max          GraphInfo      // upper select bounds
}

// PrintOpts specifies what is printing when printing a graph
type PrintOpts struct {
	Label     string // Prefix label
	Graph     bool   // If set, prints graph construction expr
	Matrix    bool   // if set, prints matrix representation of graph
	NumTraces int    // Num of Traces to print (-1 denotes natural length, 0 denotes no traces)
	CycleSpec bool   // If set, the cycles spectrum is printed -- i.e. a canonic column of "cycles" vectors
}

// DefaultPrintOpts{}
var DefaultPrintOpts = PrintOpts{
	Graph:     true,
	NumTraces: -1,
}

type FactorCatalog interface {
	generics.RefCloser

	// Tries to add the given graph encoding to this catalog.
	// Assumes TX is being added in ascending order of VertexCount() since prime detection requires all primes of lesser vertex count to have already been added.
	TryAddTraces(TX TracesProvider) (TracesID, bool)

	// NumTraces returns the number of Traces in this catalog for a given vertex count.
	// An out of bounds vertex count returns 0.
	NumTraces(forVtxCount byte) int64

	// Emits all factorizations of the given Traces using a dynamic programming algorithm to traverse all possible TX partitions.
	FindFactorizations(TX TracesProvider) <-chan FactorSet
}

// FactorSet is a set of FactorRuns, sorted by ID
type FactorSet []FactorRun

type FactorRun struct {
	ID    TracesID
	Count uint32
}
