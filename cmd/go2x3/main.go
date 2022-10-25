package main

import (
	"flag"

	"github.com/plan-systems/klog"
)

func main() {

	flag.Set("logtostderr", "true")
	flag.Set("v", "2")

	fset := flag.NewFlagSet("", flag.ContinueOnError)
	klog.InitFlags(fset)
	fset.Set("logtostderr", "true")
	fset.Set("v", "2")
	klog.SetFormatter(&klog.FmtConstWidth{
		FileNameCharWidth: 16,
		UseColor:          true,
	})

	// "github.com/alecthomas/kong"
	// ctx := kong.Parse(&cli)

	flag.Parse()

	pathname := flag.Arg(0)
	go_gpython(pathname)

	klog.Flush()
}
