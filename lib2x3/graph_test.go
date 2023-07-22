package lib2x3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/2x3systems/go2x3/go2x3"
	"github.com/2x3systems/go2x3/lib2x3"
)


func TestBasics(t *testing.T) {

	PrintCycles("1^^-2-3~4^^,3-5")
	
	PrintCycles("1-2-3=4-5=6-7=8-1")
	PrintCycles("1^-2-3-4-2,1-4") // K4 v2

	PrintCycles("1-2=3-4=5")
	
	PrintCycles("1---2")
	PrintCycles("1--~2")
	PrintCycles("1-~~2")
	PrintCycles("1~~~2")

	PrintCycles("1~2~3-1-4-5-2,3-6~7~4,5-8~6,7-8")
	
	PrintCycles("1^-2~3~6-7^-8~5~4-2,6-8,1-4")                         //

	PrintCycles("1^-~2~3~~4")
	

	PrintCycles("1~2-3")

	PrintCycles("1^-~2-3=4")

	PrintCycles("1-2,2=3")
	PrintCycles("1-2,1-4,2-3,2-4,3-4")

	PrintCycles("1~2-3~1-4-5-2,3-6-7~4,5-8-6,7-8")                         //
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
	X.WriteAsString(&b, go2x3.PrintOpts{
		Graph:     true,
		Matrix:    true,
		NumTraces: 10,
		CycleSpec: true,
	})
	str := b.String()
	fmt.Println(str)
	X.Reclaim()
}
