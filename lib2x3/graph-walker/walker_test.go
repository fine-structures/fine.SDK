package walker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fine-structures/fine.SDK/go2x3"
)

func TestEnum(t *testing.T) {
	stream, err := EnumPureParticles(EnumOpts{
		VertexMax: 8,
	})
	if err != nil {
		t.Fatal(err)
	}

	buf := strings.Builder{}
	buf.Grow(256)

	printOpts := go2x3.PrintOpts{
		NumTraces: 12,
	}

	rowCount := 0
	for X := range stream.Outlet {
		rowCount++; fmt.Fprintf(&buf, "%06d,", rowCount)
		X.WriteCSV(&buf, printOpts)
		buf.WriteByte('\n')
		fmt.Printf("%s", buf.String())
		
		buf.Reset()
		X.Reclaim()
	}

}
