package graph

import (
	"errors"
)

// Errors
var (
	ErrUnmarshal          = errors.New("unmarshal failed")
	ErrBadCatalogParam    = errors.New("bad catalog param")
	ErrInsufficientTraces = errors.New("insufficient traces")
)

// TracesID uniquely identifies a Graph's trace spectrum ("Traces")
// The most significant byte is the number of vertices, and the lower bytes are the "series ID", identifying the Traces
type TracesID uint64

// // VtxID is one-based index that identifies a vertex in a given graph (1..VtxMax)
// type VtxID byte

const (

	// VtxMax is the max possible value of a VtxID (a one-based index).
	MaxVtxID = 31

	// VtxIDBits is the number of bits dedicated for a VtxID.  It must be enough bits to represent MaxVtxID.
	VtxIDBits byte = 5

	// // VtxIDMask is the corresponding bit mask for a VtxID
	// VtxIDMask VtxID = (1 << VtxIDBits) - 1
	
)


type ExportOpts int32

const (
	ExportAsAscii ExportOpts = 1 << iota
	ExportGraphState
)

type TracesProvider interface {
    NumVertices() int
    Traces(numTraces int) Traces
}


type GraphState interface {
    TracesProvider
    
    
    ExportStateEncoding(io []byte, opts ExportOpts) ([]byte, error)
}

/*
//func ExportTracesKey(X TracesProvider

// Appends this graph's traces and canonic signature to the given buffer:
//
//	Nv + varint([Nv], NUL, NUL) + GraphUID
func ExportGraphEncoding(X GraphState, numTraces int, opts ExportOpts, io []byte) (tracesKey, graphKey []byte, err error) {
//	Nv := X.NumVerts()

	TX := X.Traces(numTraces)
	if len(TX) == 0 || len(TX) < int(numTraces) {
	    return nil, nil, ErrInsufficientTraces
    }
    
	tracesKey = TX.AppendOddEvenEncoding(io)
	var completeKey []byte
	
    if opts & ExportGraphState != 0 {
	    //key = append(key, byte(Nv)) // needed?
	    tracesKey = append(tracesKey, 0, 0) // needed??
	    
	    completeKey, err = X.ExportStateEncoding(tracesKey, opts)
	    if err != nil {
	        return nil, nil, err
        }
    }

	return tracesKey, completeKey, nil
}
*/


// // Appends this graph's traces and canonic signature to the given buffer:
// //
// //	Nv + varint([Nv], NUL, NUL) + GraphUID
// func FormExtendedKey(X GraphState, numTraces int, io []byte, opts ExportOpts) (tracesKey, graphKey []byte) {
// 	key := append(in, X.NumVerts())
// 	key = X.Traces(0).AppendTraceSpecTo(key)
// 	key = append(key, 0, 0)

// 	full := X.xstate.ExportEncoding(key, opts)
// 	return key, full
// }


// type EdgeTraces struct {
// 	NetCount int64
// 	TracesID int64 
// }

// type ParticleState struct {
// 	Traces1 EdgeTraces // Odd component of the particle's Traces
// 	Traces2 EdgeTraces // Even component of the particle's Traces
// 	Edges   []*VtxEdge // Optional -- constituent edges that produce Odd and Even
// }


var (
	ErrBadEncoding   = errors.New("bad graph encoding")
	ErrBadVtxID      = errors.New("bad graph vertex ID")
	ErrMissingVtxID  = errors.New("missing vertex ID")
	ErrBadEdge       = errors.New("bad graph edge")
	ErrBadEdgeType   = errors.New("bad graph edge type")
	ErrBrokenEdges   = errors.New("bad or inconsistent graph edge configuration")
	ErrViolates2x3   = errors.New("graph is not a valid 2x3")
	ErrVtxExpected   = errors.New("vertex ID expected")
	ErrSitesExceeded = errors.New("number of loops and edges exceeds 3")
	ErrNilGraph      = errors.New("nil graph")
	ErrInvalidVtxID  = errors.New("invalid vertex or group ID")
)

