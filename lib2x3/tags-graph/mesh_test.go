package tags_graph

import (
	"fmt"
	"testing"
)

func TestGraphsParsing(t *testing.T) {
	testExprs := []string{
		"weak boson", "(+ - +)",         // weak boson
		"gamma boson", "() (+1 +1 +1)",   // gamma boson
		"() (+1 +1 +1)^3", // Higgs boson
	}
	sb := make([]byte, 0, 1024)
	
	for _, expr := range testExprs {
		X, err := ParseMeshExpr(expr)
		if err != nil {
			t.Error(err)
		}
		for _, vi := range X.Roots {
			var cout []byte
			cout, err = vi.MarshalOut(sb[:0], AsCSV)
			if err != nil {
				t.Error(err)
			}
			fmt.Println(string(cout))
		}
	}
	
	//X, err := ParseMeshExpr("(+-+)")
	X, err := ParseMeshExpr("() (+1 +1 +1)^3") // higgs
	//X, err := ParseMeshExpr("(xxx+(o+(x(xo(xxoxx)x)x-)-(oo)))")
	// X, err := ParseMeshExpr("(xxx)")
	//X, err := ParseMeshExpr("(x+o(xx))")
	if err != nil {
		t.Error(err)
	}
	X, err = ParseMeshExpr("((ox)-(o+)+)")
	if err != nil {
		t.Error(err)
	}
	// X, err = ParseMeshExpr("(.(+(---)=)(...))")
	// if err != nil {
	// 	t.Error(err)
	// }
	fmt.Println(X)
}
