package main

import (
	"fmt"
	"time"

	"log"

	"github.com/go-python/gpython/py"
	"github.com/go-python/gpython/repl"
	"github.com/go-python/gpython/repl/cli"

	_ "github.com/fine-structures/fine.SDK/py2x3"
	_ "github.com/go-python/gpython/stdlib"
)

func go_gpython(pathname string) {
	ctx := py.NewContext(py.DefaultContextOpts())

	var (
		err error
	)
	if len(pathname) == 0 {
		replCtx := repl.New(ctx)

		_, err = py.RunFile(ctx, "lib/_REPL_startup.py", py.CompileOpts{}, replCtx.Module)
		if err == nil {
			cli.RunREPL(replCtx)
		}

	} else {
		startTime := time.Now()
		fmt.Printf("<<<>>>   executing '%s'   <<<>>>\n", pathname)

		_, err = py.RunFile(ctx, pathname, py.CompileOpts{}, nil)

		if err == nil {
			t := time.Now()
			elapsed := t.Sub(startTime)
			fmt.Printf("<<<>>>   execution complete: %v   <<<>>>\n", elapsed)
		}

	}

	ctx.Close()
	<-ctx.Done()

	if err != nil {
		py.TracebackDump(err)
		log.Fatal(err)
	}

}
