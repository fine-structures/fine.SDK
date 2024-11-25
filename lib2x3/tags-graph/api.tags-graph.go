package tags_graph

// package orca
// ordinal.recursive.counting.algorithm
// spectacle-tagging-system
// package STS

type Group struct {
	Roots []Vertex `(@@ (";" @@)*)?`
}

const MeshGroupCharacterSet = "[]"

type Vertex struct {
	Edges []Edge `@@*`
}

type Edge struct {
	Direction int64  // cardinal direction: 0, 1,-1, 2,-2, 3,-3, .. 2^63,-2^63
	Sign      int64  // weight or count:    0, 1,-1, 2,-2, 3,-3, .. 2^63,-2^63
	Op        OpCode // which operation to perform
	Vertex    Vertex `( "(" @@ ")" )`
	Symbol    string `| @( ( "+" | "-" )? @Symbol )`
}

type OpCode byte

const (
	OpCode_Sprout           OpCode = 'o'
	OpCode_SproutInverse    OpCode = 'x'
	OpCode_Duplicate        OpCode = '+'
	OpCode_DuplicateInverse OpCode = '-'
)

func ParseMeshExpr(expr string) (*Group, error) {
	ast, err := sParseGraphsExpr.ParseString("", expr)
	if err != nil {
		return nil, err
	}
	err = ast.Validate(ValidateParse)
	if err != nil {
		return nil, err
	}
	return ast, nil
}

// Format:
//   <int>*( )(POW

type TagLiteral struct {
	Count         *int64      `  @(Int)`
	Number        *float64    `  @(Float|Int)`
	Literal       *string     `| @(Ident | "+" | "-")`
	Subexpression *Expression `| "(" @@ ")"`
}

// type SignedValue struct {
// 	Operator Operator `@("+" | "-")`
// 	Term     *Term    `@@`
// }

// type Factor struct {
// 	Base     *Value `@@`
// 	Exponent *Value `( "^" @@ )?`
// }

// type OpFactor struct {
// 	Operator Operator `@("*" | "/")`
// 	Factor   *Factor  `@@`
// }

// type Term struct {
// }

type OpTerm struct {
	Count     *int64  `( ( @Int     )`
	Direction int64   `  ( "@" @Int )?`
	Label     *string `    ( @Ident | `
	Symbol    *string `               @("." | "~" | "+" | "-") )`

	Value TagLiteral `@@*`
}

type Expression struct {
	Left  *OpTerm   `@@`
	Right []*OpTerm `@@*`
}
