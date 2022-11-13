package lib2x3

import (
	"errors"

	"github.com/go-python/gpython/py"
)

// TracesID uniquely identifies a Graph's trace spectrum ("Traces")
// The most significant byte is the number of vertices, and the lower bytes are the "series ID", identifying the Traces
type TracesID uint64

// OnGraphHit is a callback proc used to return Graph's meeting a set of selection criteria.
// Ownership of a Graph also travels through the channel.
type OnGraphHit chan<- *Graph

// Errors
var (
	ErrUnmarshal          = errors.New("unmarshal failed")
	ErrBadCatalogFilename = errors.New("bad catalog filename")
)

// NewCatalogContext is a forward declared entry point, allowing Catalog implementations to decouple from the lib2x3 module.
var NewCatalogContext func() CatalogContext

// CatalogContext is a container for open / active Catalog instances.
type CatalogContext interface {

	// Opens a new or existing 2x3 particle catalog for access in this workspace
	OpenCatalog(opts CatalogOpts) (Catalog, error)

	// Closes all open catalogs to be closed then closes.
	Close()

	// Signals when Close() completed and all open Catalogs have been closed
	Done() <-chan struct{}
}

// CatalogOpts specifies params for opening a lib2x3 Catalog
type CatalogOpts struct {
	DbPathName string
	ReadOnly   bool
	TraceCount int32
	NeedPrimes bool
}

// Catalog wraps a database of lib2x3 Graph encodings.
type Catalog interface {

	// Tries to add the given graph encoding to this catalog.
	// If true is returned, X did not exist, was added, and X.TracesID is set to the newly issued TracesID.
	TryAddGraph(X *Graph) bool

	// Returns true if this catalog was opened for read-only access.
	IsReadOnly() bool

	// NumTraces returns the number of particle primes for a given vertex count (one-based indexing).
	// NumTraces()[0] is always 0 and an out of bounds vertex count returns 0.
	NumTraces(forVtxCount byte) int64

	// NumPrimes returns the number of particle primes for a given vertex count (one-based indexing).
	// NumPrimes()[0] is always 0 and an out of bounds vertex count returns 0.
	NumPrimes(forVtxCount byte) int64

	// Type returns info for gpython support
	Type() *py.Type

	// Select fires the given callback with each GraphEncoding that meets the selection criteria.
	Select(sel GraphSelector, onHit OnGraphHit)

	// Closes this catalog
	Close()
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

// GraphSelector is an operator that either selects a given Graph or not.
type GraphSelector struct {
	Traces *Graph // Implies a Traces to match with or factor
	//Traces       Traces    // len(Traces) > 0 implies a Traces to match with or factor
	//TracesNv     int       // NumVerts associated with Traces
	Factor       bool      // Perform factorization of sel.Traces
	UniqueTraces bool      // Only select the first Graph for each unique traces
	PrimesOnly   bool      // Only select prime traces
	Min          GraphInfo // lower select bounds
	Max          GraphInfo // upper select bounds
}

// AllowGraph is a convenience function used to see if a Graph is selected according to a GraphSelector.
func (sel *GraphSelector) AllowGraph(X *Graph) bool {
	info := X.GetInfo()
	if info.NumParticles < sel.Min.NumParticles || info.NumVerts < sel.Min.NumVerts || info.PosLoops < sel.Min.PosLoops || info.NegLoops < sel.Min.NegLoops || info.PosEdges < sel.Min.PosEdges || info.NegEdges < sel.Min.NegEdges {
		return false
	}
	if info.NumParticles > sel.Max.NumParticles || info.NumVerts > sel.Max.NumVerts || info.PosLoops > sel.Max.PosLoops || info.NegLoops > sel.Max.NegLoops || info.PosEdges > sel.Max.PosEdges || info.NegEdges > sel.Max.NegEdges {
		return false
	}
	return true
}
