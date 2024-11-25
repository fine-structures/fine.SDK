package tags_graph

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func (op *OpCode) AsCSV() byte {
	switch *op {
	case OpCode_Sprout:
		return 'o'
	case OpCode_SproutInverse:
		return 'x'
	case OpCode_Duplicate:
		return '+'
	case OpCode_DuplicateInverse:
		return '-'
	}
	return '?'
}

type MarshalOpts int32

const (
	AsLSM MarshalOpts = 0
	AsCSV MarshalOpts = 1
)

type Marshaller interface {
	MarshalOut(dst []byte, opts MarshalOpts) ([]byte, error)
}

const (
	U16TicksPerSecond = int64(1 << 16)
	U16TicksPerHour   = int64(60 * 60 * U16TicksPerSecond)
	U16TicksPerDay    = int64(24 * U16TicksPerHour)

	Head_000 = 0
	Head_030 = U16TicksPerDay / 12
	Head_045 = U16TicksPerDay / 8
	Head_060 = U16TicksPerDay / 6
	Head_090 = U16TicksPerDay / 4
	Head_120 = U16TicksPerDay / 3
	Head_180 = U16TicksPerDay / 2
	Head_240 = U16TicksPerDay * 2 / 3
	Head_270 = U16TicksPerDay * 3 / 4
	Head_300 = U16TicksPerDay * 5 / 6
	Head_315 = U16TicksPerDay * 7 / 8
	Head_330 = U16TicksPerDay * 11 / 12
	Head_360 = U16TicksPerDay
)

type ValidateOp int32

const (
	ValidateParse  ValidateOp = 0
	ValidateAsRoot ValidateOp = 1
)

var sGraphLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Punct", `[();]`},
	{"Sign", `[-+]`},
	{"Symbol", `[A-Za-z0-9]`},
	{"whitespace", `[ \t]+`},
})

var sParseGraphsExpr = participle.MustBuild[Group](
	participle.Lexer(sGraphLexer),
)

func (group *Group) Validate(op ValidateOp) error {
	for _, root := range group.Roots {
		if err := root.Validate(op | ValidateAsRoot); err != nil {
			return err
		}
	}
	return nil
}

func (group *Group) MarshalOut(dst []byte, opts MarshalOpts) ([]byte, error) {
	for _, root := range group.Roots {
		root.MarshalOut(dst, opts)
	}
	return dst, nil
}

func (v *Vertex) Validate(op ValidateOp) error {
	for _, ei := range v.Edges {
		if err := ei.Validate(op); err != nil {
			return err
		}
		// for _, sub := range ei.To.Edges {
		// 	if err := sub.Validate(op &^  ); err != nil {
		// 		return err
		// 	}
		// }
	}
	return nil
}

func (v *Vertex) MarshalOut(dst []byte, opts MarshalOpts) ([]byte, error) {
	switch opts {
	case AsCSV:
		dst = append(dst, '(')
		for _, ei := range v.Edges {
			ei.MarshalOut(dst, opts)
		}
		dst = append(dst, ')')
	}
	return dst, nil
}

func (edge *Edge) Validate(op ValidateOp) error {
	// switch edge.Symbol {
	// case "o":
	// 	edge.Op = OpCode_SproutEdge
	// 	edge.Flow = +1
	// case "x":
	// 	edge.Op = OpCode_SproutEdge
	// 	edge.Flow = -1
	// case "+":
	// 	edge.Op = OpCode_AddEdgeEdge
	// 	edge.Flow = +1
	// case "-":
	// 	edge.Op = OpCode_AddEdgeEdge
	// 	edge.Flow = -1
	// }
	return nil
}

func (edge Edge) MarshalOut(dst []byte, opts MarshalOpts) ([]byte, error) {
	if edge.Sign != 0 {
		if len(edge.Vertex.Edges) > 0 {

		} else {
			dst = append(dst, edge.Op.AsCSV())
		}
	}

	switch opts {
	case AsCSV:
		dst = append(dst, edge.AsCSV())
	}
	return dst, nil
}

func (edge *Edge) ApplyToken(token rune) {
	var sign int64
	var cmd OpCode

	switch token {

	case
		'.',
		'o',
		'O',
		0x00B7, // ·
		0x0387, // ·
		0x05F3, // ׳
		0x2022, // •
		0x00BA, // º
		0x09FD, // ৽
		0x0970, // ॰
		0x0AF0, // ૰
		0x25E6: // ◦
		sign = +1
		cmd = OpCode_Sprout

	case
		'x',
		'X':
		sign = -1
		cmd = OpCode_Sprout

	case
		'+':
		sign = +1
		cmd = OpCode_Duplicate

	case
		'-':
		sign = -1
		cmd = OpCode_Duplicate
	}

	edge.Sign = sign
	edge.Op = cmd
}

func (edge Edge) AsCSV() byte {
	t_pos := byte('?')
	t_neg := byte('?')

	switch edge.Op {
	case
		OpCode_Sprout:
		t_pos = 'o'
		t_neg = 'x'
	case
		OpCode_Duplicate:
		t_pos = '+'
		t_neg = '-'
	}

	asc := byte(' ')
	if edge.Sign > 0 {
		asc = t_pos
	} else if edge.Sign < 0 {
		asc = t_neg
	}
	return asc
}
