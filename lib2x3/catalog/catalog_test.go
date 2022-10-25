package catalog_test

import (
	"os"
	"path"
	"testing"

	"github.com/2x3systems/go2x3/lib2x3"
)

var primes = []string{
	"1", "1^", "1^^", "1^^^", // v1
	"1=2^", "1^^-2^", "1-2^", // v2

}

var (
	gT *testing.T

	gWorkspace = &lib2x3.Workspace{
		CatalogCtx: lib2x3.NewCatalogContext(),
	}
)

func TestBasics(t *testing.T) {

	gT = t
	dir, err := os.MkdirTemp("", "junk*")
	if err != nil {
		gT.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := lib2x3.CatalogOpts{
		NeedPrimes: true,
		DbPathName: path.Join(dir, "TestBasics"),
	}
	cat, err := gWorkspace.CatalogCtx.OpenCatalog(opts)
	if err != nil {
		gT.Fatal(err)
	}
	defer cat.Close()

	X := lib2x3.NewGraph(nil)

	for _, Xstr := range primes {
		X.InitFromString(Xstr)
		if added := cat.TryAddGraph(X); !added {
			t.Fatal("nope")
		}
		if added := cat.TryAddGraph(X); added {
			t.Fatal("nope")
		}
	}

	// Add known non-prime
	X.InitFromString("1-2")
	if added := cat.TryAddGraph(X); !added {
		t.Fatal("nope")
	}

	// Add a known prime
	X.InitFromString("1~2,1~3,2-3")
	if added := cat.TryAddGraph(X); !added {
		t.Fatal("nope")
	}

	// Select -- we should get all the particles we've added so far
	{
		total := 0
		onHit := make(chan *lib2x3.Graph)
		go func() {
			cat.Select(lib2x3.DefaultGraphSelector, onHit)
			close(onHit)
		}()
		for X := range onHit {
			X.Println(">>>")
			total++
		}
		if total != 9 {
			t.Fatal("Select fail")
		}
	}

	// Factor a photon -- should get e + ~e
	{
		Xsrc := lib2x3.NewGraph(nil)
		Xsrc.InitFromString("1---2")

		sel := lib2x3.DefaultGraphSelector
		sel.Factor = true
		sel.Traces = Xsrc

		total := 0
		onHit := make(chan *lib2x3.Graph)
		go func() {
			cat.Select(sel, onHit)
			close(onHit)
		}()
		for X := range onHit {
			X.Println(">>>")
			total++
			if !X.Traces(10).IsEqual(Xsrc.Traces(10)) {
				t.Fatal("traces don't match")
			}
		}
		if total != 1 {
			t.Fatal("factorization fail")
		}
	}
}
