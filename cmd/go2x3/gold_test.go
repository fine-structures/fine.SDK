package main

import (
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/go-python/gpython/py"
)

func TestGold(t *testing.T) {
	err := os.Chdir("./")
	if err != nil {
		log.Fatal(err)
	}

	scriptDir := "learn/"
	files, err := os.ReadDir(scriptDir)
	if err != nil {
		log.Fatal(err)
	}

	goldDir := path.Join(scriptDir, "gold")
	os.MkdirAll(goldDir, 0700)

	for _, fi := range files {
		pyFile := path.Join(scriptDir, fi.Name())
		ext := filepath.Ext(pyFile)
		if ext != ".py" {
			continue
		}
		// if !strings.HasPrefix(pyFile, "learn/05") {
		// 	continue
		// }

		outputPathname := path.Join(goldDir, pyFile[len(scriptDir):len(pyFile)-len(ext)]+".txt")
		{
			ctx := py.NewContext(py.DefaultContextOpts())
			redirect, err := RedirectToFile(outputPathname, ctx)
			if err != nil {
				log.Fatal(err)
			}

			_, err = py.RunFile(ctx, pyFile, py.CompileOpts{}, nil)
			if err != nil {
				log.Fatal(err)
			}
			ctx.Close()
			<-ctx.Done()

			if err = redirect.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}
}

type pyRedirect struct {
	file       *os.File
	prevStdout *os.File
}

func RedirectToFile(outputPathname string, ctx py.Context) (io.Closer, error) {
	ofile, err := os.OpenFile(outputPathname, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}

	sys := ctx.Store().MustGetModule("sys")
	sys.Globals["stdout"] = &py.File{
		File:     ofile,
		FileMode: py.FileWrite,
	}

	redir := &pyRedirect{
		file:       ofile,
		prevStdout: os.Stdout,
	}

	os.Stdout = ofile

	// MultiWriter writes to saved stdout and file
	// mw := io.MultiWriter(actual_stdout, f)

	// // get pipe reader and writer | writes to pipe writer come out pipe reader
	// r, w, _ := os.Pipe()

	// replace stdout,stderr with pipe writer | all writes to stdout, stderr will go through pipe instead (f
	//os.Stdout = w

	return redir, nil
}

func (redir *pyRedirect) Close() error {
	if redir.prevStdout == nil {
		return nil
	}

	// Restore the previous Stdout and close the output file
	os.Stdout = redir.prevStdout
	err := redir.file.Close()
	redir.file = nil
	return err
}
