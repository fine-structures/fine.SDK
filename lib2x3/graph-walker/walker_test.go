package walker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fine-structures/fine-sdk-go/go2x3"
)

func TestEnum(t *testing.T) {
	stream, err := EnumPureParticles(EnumOpts{
		VertexMax: 6,
	})
	if err != nil {
		t.Fatal(err)
	}

	buf := strings.Builder{}
	buf.Grow(256)
	count := 0

	printOpts := go2x3.PrintOpts{
		NumTraces: 12,
	}

	for X := range stream.Outlet {
		count++
		fmt.Fprintf(&buf, "%06d,", count)
		X.WriteAsString(&buf, printOpts)

		buf.WriteByte('\n')
		fmt.Printf("%s", buf.String())
		buf.Reset()

		X.Reclaim()
	}

}
