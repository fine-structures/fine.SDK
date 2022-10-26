package lib2x3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/2x3systems/go2x3/lib2x3"
)

func TestMisc(t *testing.T) {
	tmp := &strings.Builder{}

	N5 :=
		"0 1 2 2 1 \n" +
			"1 0 1 2 2 \n" +
			"2 1 0 1 2 \n" +
			"2 2 1 0 1 \n" +
			"1 2 2 1 0 \n"

	N6 :=
		"0 1 2 3 2 1 \n" +
			"1 0 1 2 3 2 \n" +
			"2 1 0 1 2 3 \n" +
			"3 2 1 0 1 2 \n" +
			"2 3 2 1 0 1 \n" +
			"1 2 3 2 1 0 \n"

	for Nv := int32(2); Nv <= 6; Nv++ {
		tmp.Reset()
		for j := int32(0); j < Nv; j++ {
			for i := int32(0); i < Nv; i++ {
				dist := lib2x3.ShortestEdgeDist(Nv, i, j)
				fmt.Fprintf(tmp, "%d ", dist)
			}
			fmt.Fprint(tmp, "\n")
		}
		out := tmp.String()
		t.Logf("Nv = %v:\n%s\n", Nv, out)
		if Nv == 5 && out != N5 {
			t.Fatalf("Nv = 5 test failed, should be:\n%s\n", N5)
		}
		if Nv == 6 && out != N6 {
			t.Fatalf("Nv = 6 test failed, should be:\n%s\n", N6)
		}
	}

}

func TestBasics(t *testing.T) {

	PrintCycles("1-2-3")

	PrintCycles("1-2,2=3")
	PrintCycles("1-2,1-4,2-3,2-4,3-4")

	// PrintCycles("1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8")                         //
	// PrintCycles("1-2, 1-3, 1-4, 2-5, 4-5, 2-6, 3-6, 3-7, 4-7, 5-8, 6-8, 7-8") // 001
	// PrintCycles("1~2, 1~3, 1~4, 2-5, 3-5, 2-6, 4-6, 3-7, 4-7, 5~8, 6~8, 7~8") // 002
	// PrintCycles("1~2, 1~3, 1~4, 2-5, 4-5, 2-6, 3-6, 3-7, 4-7, 5-8, 6-8, 7-8") // 003
	// PrintCycles("1-2, 1-3, 1-4, 2-5, 3-5, 2-6, 4-6, 3~7, 4~7, 7-8, 5~8, 6~8") // 004
	// PrintCycles("1-2, 1-3, 1~4, 4-5, 3~5, 2-6, 4~6, 2~7, 3~7, 5-8, 7-8, 6~8") // 010
	// X.Println("higgs: ")

	// calc := GetGraphTool()
	// calc.MakeCanonic(X)
	// calc.Reclaim()

	// calc := GetGraphTool()
	// calc.MakeCanonic(X)
	// calc.Reclaim()

}

func PrintCycles(Xstr string) {
	X := lib2x3.NewGraph(nil)
	X.InitFromString(Xstr)

	b := strings.Builder{}
	b.Grow(192)
	X.WriteAsString(&b, lib2x3.PrintOpts{
		Graph:     true,
		Matrix:    true,
		NumTraces: 10,
		Tricodes:  true,
		CycleSpec: true,
	})
	str := b.String()
	fmt.Println(str)

}
