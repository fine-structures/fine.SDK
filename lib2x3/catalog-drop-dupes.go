package lib2x3

import (
	"bytes"
	"hash/maphash"

	"github.com/2x3systems/go2x3/lib2x3/graph"
)

type dropDupes struct {
	hashMap   map[uint64]GraphEncoding
	hasher    maphash.Hash
	bufPool   []byte
	bufPoolSz int
	opts      DropDupeOpts
}

const DefaultPoolSz = 32 * 1024

type DropDupeOpts struct {
	PoolSz        int  // 0 denotes DefaultPoolSz (32k)
	//UseTracesOnly bool // if set, two graphs with equal traces are considered equal (vs equivalent graphs)
}

func NewDropDupes(opts DropDupeOpts) GraphAdder {
	if opts.PoolSz <= 0 {
		opts.PoolSz = DefaultPoolSz
	}
	return &dropDupes{
		hashMap: make(map[uint64]GraphEncoding),
		opts:    opts,
	}
}

func (cat *dropDupes) Reset() {
	cat.bufPoolSz = 0
	for k := range cat.hashMap {
		delete(cat.hashMap, k)
	}
}

func (cat *dropDupes) Close() {
	cat.Reset()
	cat.hashMap = nil
}

func (cat *dropDupes) TryAddGraph(X *Graph) bool {
	var keyBuf [512]byte
	tracesKey := X.Traces(0).AppendOddEvenEncoding(keyBuf[:0])
	Xkey, _ := X.ExportStateEncoding(tracesKey, graph.ExportGraphState)

	cat.hasher.Reset()
	cat.hasher.Write(Xkey)
	hash := cat.hasher.Sum64()

	existing, found := cat.hashMap[hash]
	for found {
		if bytes.Equal(existing, Xkey) {
			return false
		}
		hash++
		existing, found = cat.hashMap[hash]
	}

	// If we've gotten here, it means this is a new entry.
	// Place a copy of the buf in our backing buf (in the heap).
	// If we run out of space in our pool, we start a new pool
	pos := cat.bufPoolSz
	itemLen := len(Xkey)
	if pos+itemLen > cap(cat.bufPool) {
		allocSz := max(cat.opts.PoolSz, itemLen)
		cat.bufPool = make([]byte, allocSz)
		cat.bufPoolSz = 0
		pos = 0
	}

	// Place the backed copy of the graph ID buf at the open hash spot
	cat.hashMap[hash] = append(cat.bufPool[pos:pos], Xkey...)
	cat.bufPoolSz += itemLen
	return true
}
