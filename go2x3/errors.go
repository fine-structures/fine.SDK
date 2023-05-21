package go2x3

import "errors"

// Errors
var (
	ErrUnmarshal          = errors.New("unmarshal failed")
	ErrBadCatalogParam    = errors.New("bad catalog param")
	ErrInsufficientTraces = errors.New("insufficient traces")
	ErrBadEncoding        = errors.New("bad graph encoding")
	ErrBadVtxID           = errors.New("bad graph vertex ID")
	ErrMissingVtxID       = errors.New("missing vertex ID")
	ErrBadEdge            = errors.New("bad graph edge")
	ErrBadEdgeType        = errors.New("bad graph edge type")
	ErrBrokenEdges        = errors.New("bad or inconsistent graph edge configuration")
	ErrViolates2x3        = errors.New("graph is not a valid 2x3")
	ErrVtxExpected        = errors.New("vertex ID expected")
	ErrSitesExceeded      = errors.New("number of loops and edges exceeds 3")
	ErrNilGraph           = errors.New("nil graph")
	ErrInvalidVtxID       = errors.New("invalid vertex or group ID")
)
