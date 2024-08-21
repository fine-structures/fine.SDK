package graph

// CatalogID is a unique identifier for any valid "2x3" graph.
//
// It is an enumeration sequence index, meaning that any graph is a unique number of steps "away" from any other graph.
//
// What the Wolfram project misses is that as a graph grows, is will pass through states such that a coefficient integer "odd even" pair
// can be factored out, where the pair (1,1) is the identity.  This integer pair can be regarded as the count or amplitude of the odd and even Traces terms.
//
// So if (a,b) is a factor, then (n*a, n*b) is the next factor.  This is the same as the Fibonacci sequence, but with a twist.
type CatalogID [10]byte

/*
type Graph interface {
	go2x3.TracesProvider

	Init()

	// Parses the given graph expr, assigns this graph that state,
	// calculates the traces, looks up the CatalogID for the traces, and assigns that to this graph.
	InitFromGraphExpr(expr string) error

	GrowFromSteps(steps []fine.GrowOp) error

	Export() Encodings

	AddRef()
	ReleaseRef()
}


func New() Graph {
	return nil // TODO
}

type Encodings struct {
	//EncodingType   byte // base 8 octal or ascii symbols

	String        []byte
	EnumerationID []byte
	Ops           []GrowOp
}
*/

// func FromExpression(expr string) (GraphVM, error) {
// 	return nil, nil
// }

// func (ge GraphEncoding) AppendOps() {

// }

type GraphBuilder interface {
}

type GraphWalker interface {
	Reset()

	EmitNextGraph() (VtxGraphVM, error)
}
