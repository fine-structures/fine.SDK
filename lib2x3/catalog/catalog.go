package catalog

import (
	"bytes"
	"runtime"

	"github.com/astronomical-grace/fine-structures-go/go2x3"
	"github.com/astronomical-grace/fine-structures-go/lib2x3"
	"github.com/astronomical-grace/fine-structures-go/lib2x3/factor"
	"github.com/astronomical-grace/fine-structures-go/lib2x3/graph"
	"github.com/pkg/errors"

	"github.com/dgraph-io/badger/v4"
)

/***

Catalog database format:





	type TracesSpec := [Oi]varint, [Ei]varint, [±] (byte)

	gCatalogStateKey => CatalogState


	TracesSpec, NUL, NUL (UserMeta uses kIsPrime flag)
		CanonicStateEncoding ([]byte hash)        => GraphDef
		...
	...


	gCatalogTracesKey


	gPrimesCatalogKey (EdgePrimeTracesID <-> TracesLSM)
			TracesLSM  (EdgePrimeTraces)
			..

	TracesLSM, NUL, NUL,
		CanonicStateEncoding  => GraphSpecDef
		...
	...



The above structure allows to:
	1) load all primes for a given Nv
	2) enumerate all Graphs (in a somewhat predictable order) for a given Nv
	3) check if a given Traces or Graph has been added

Currently, the a downside with the above is that to read in all the primes requires a complete walk through all graphs,
	so loading all the primes through v=8 takes about a second and takes several seconds for v=10.

Next steps / ideas:
	- Build primes-only table:
		- Step 1: has a given Traces been witnessed yet? (i.e. has a prime test already been performed)
		- Step 2: if witnessed then skip; otherwise, perform prime test, add prime entry if prime, and update Traces as witnessed.
		- advantages:
			- since it only contains primes, space won't be consumed by non-prime encodings
			- reading in primes is FAST
			- NumPrimes() goes away and only NumTraces() remains
			- suitable location to assign a preferred GraphDef for a given prime
	- Maintain a prime catalog separate from an encodings catalog
		- This speeds finding all primes (v=10 is almost a billion encodings -- all go unused)
		- The prime catalog only keeps the best preferred graph encoding (has designated "prime encoding")
		- Each catalog would be auto-generating

	- Functionality that generates all possible product combos given any p >= 1 Graph

	- Compute "commutator" matrix given two X1 and X2 (given to be same traces Graphs)
		- Xc12: X1 = Xc12 X2
		- Xc21: X2 = Xc21 X1
		- where I = Xc21 Xc12 = Xc12 Xc21
		commutator: the transformation matrix that transforms

	- Encode TracesSpec as a varuint64 followed by a sign (bit) vector (or a way to represent negating graph adjacency matrix)

	- Can generatign functions be complete and canonical?  Something more compact than:
		Nv (byte)
			C_Cycles ([Nv][Nv]varint, NUL, NUL)
				V_Cycles ([Nv][Nv]varint)    => GraphSpec


***/

var (
	gCatalogStateKey = []byte{0x00, 0x00, 0x01}
)

// Catalog is a db wrapper for a 2x3 particle catalog
type catalog struct {
	ctx          go2x3.CatalogContext
	readOnly     bool
	stateDirty   bool
	state        graph.CatalogState
	db           *badger.DB
	CatalogDesig string
	primeCache   *factor.FactorCatalog

	// LSM double-lookup of a TracesID table:
	//   TracesID <=> [1..TracesCount/2]varint64
	//EdgeTrace symbol.Table
}

func OpenCatalog(ctx go2x3.CatalogContext, opts go2x3.CatalogOpts) (go2x3.Catalog, error) {

	if opts.TraceCount <= 0 {
		opts.TraceCount = 12
		//return nil, errors.Wrap(lib2x3.ErrBadCatalogParam, "TraceCount must be > 0")
	}

	cat := &catalog{
		ctx:          ctx,
		CatalogDesig: "B1",
	}

	dbOpts := badger.DefaultOptions(opts.DbPathName)
	dbOpts.ReadOnly = opts.ReadOnly
	dbOpts.DetectConflicts = false // not needed so disable for performance
	dbOpts.Logger = nil
	dbOpts.MetricsEnabled = false

	// Badger for windows currently does not support read-only mode
	if runtime.GOOS == "windows" {
		dbOpts.ReadOnly = false
	}

	var err error

	if len(opts.DbPathName) == 0 {
		if opts.ReadOnly {
			return nil, errors.Wrap(go2x3.ErrBadCatalogParam, "DbPathName must be specified for read-only catalog")
		}
		dbOpts.InMemory = true
	}

	cat.db, err = badger.Open(dbOpts)
	if err != nil {
		return nil, err
	}

	// Once the db is open, we consider thx catalog ctx blocked until the catalog closes
	ctx.AttachCatalog(cat)

	err = cat.loadState()
	if err == badger.ErrKeyNotFound {
		err = nil
		cat.stateDirty = true
		cat.state.MajorVers = 2022
		cat.state.MinorVers = 1
		cat.state.IsPrimeCatalog = opts.NeedPrimes
		cat.state.NumTraces = make([]uint64, opts.TraceCount+1)
		cat.state.NumPrimes = make([]uint64, opts.TraceCount+1)
		cat.state.TraceCount = opts.TraceCount
	}

	if cat.state.MajorVers != 2022 || cat.state.MinorVers != 1 {
		err = errors.New("Catalog version is incompatible")
	} else if opts.TraceCount > cat.state.TraceCount {
		err = errors.New("Catalog's TraceCount is below the requested TraceCount")
	} else if opts.NeedPrimes && !cat.state.IsPrimeCatalog {
		err = errors.New("Catalog was not created to be a prime catalog")
	}

	if err != nil {
		cat.Close()
		return nil, err
	}

	if cat.IsPrimeCatalog() {
		cat.primeCache = factor.NewFactorCatalog(cat.state.TraceCount)
	}

	return cat, nil
}

func (cat *catalog) NumTraces(forVtxCount byte) int64 {
	if forVtxCount == 0 || int(forVtxCount) > len(cat.state.NumTraces) {
		return 0
	}
	return int64(cat.state.NumTraces[forVtxCount])
}

func (cat *catalog) NumPrimes(forVtxCount byte) int64 {
	if forVtxCount == 0 || int(forVtxCount) > len(cat.state.NumPrimes) {
		return 0
	}
	return int64(cat.state.NumPrimes[forVtxCount])
}

func (cat *catalog) loadState() error {
	err := cat.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(gCatalogStateKey)
		if err == nil {
			item.Value(func(val []byte) error {
				return cat.state.Unmarshal(val)
			})
		}
		return err
	})
	return err
}

func (cat *catalog) flushState() {
	if cat.stateDirty {
		err := cat.db.Update(func(txn *badger.Txn) error {
			stateBuf, err := cat.state.Marshal()
			if err != nil {
				return err
			}
			err = txn.Set(gCatalogStateKey, stateBuf)
			if err != nil {
				return err
			}
			return err
		})
		if err != nil {
			panic(err)
		}
		cat.stateDirty = false
	}
}

func (cat *catalog) Close() error {
	cat.flushState()
	if cat.db != nil {
		cat.db.Close()
		cat.db = nil
		cat.ctx.DetachCatalog(cat)
		cat.ctx = nil
	}
	return nil
}

// TraceCount is the Traces size for each entry in the Traces catalog
func (cat *catalog) TraceCount() int {
	return int(cat.state.TraceCount)
}

func (cat *catalog) IsPrimeCatalog() bool {
	return cat.state.IsPrimeCatalog
}

func (cat *catalog) IsReadOnly() bool {
	return cat.readOnly
}

func (cat *catalog) issueNextTracesID(numVerts int) go2x3.TracesID {
	tid := cat.state.NumTraces[numVerts] + 1
	cat.state.NumTraces[numVerts] = tid
	cat.stateDirty = true

	return go2x3.FormTracesID(uint32(numVerts), tid)
}

func (cat *catalog) issueNextPrimeID(numVerts int) go2x3.TracesID {
	tid := cat.state.NumPrimes[numVerts] + 1
	cat.state.NumPrimes[numVerts] = tid
	cat.stateDirty = true

	return go2x3.FormTracesID(uint32(numVerts), tid)
}

func (cat *catalog) formCatalogKeyFromPrimeFactor(key []byte, factor go2x3.TracesID) []byte {
	Nv := uint32(factor.NumVertices())
	TX := cat.primeCache.GetFactorTraces(Nv, uint32(factor.SeriesID()))

	key = append(key, byte(Nv)) // needed?  or use edge info sorter?
	key = TX[:Nv].AppendTracesLSM(key)
	key = append(key, 0, 0)

	return key
}

func (cat *catalog) formTracesKey(key []byte, X go2x3.TracesProvider) []byte {
	Nv := X.NumVertices()
	Nt := Nv // cat.TraceCount()
	TX := X.Traces(Nt)
	// if len(TX) == 0 || len(TX) < Nt {
	//     return nil, go2x3.ErrInsufficientTraces
	// }

	key = append(key, byte(Nv))
	key = TX.AppendTracesLSM(key)
	key = append(key, 0, 0)

	return key
}

// Select will call onHit() with all graphs matching the given search criteria.
//
// Warning: if onHit() retains the given GraphEncoding, then it must make a copy.
//
// Enumeration stops when there are no more matches or if onHit() returns false.
func (cat *catalog) Select(sel go2x3.GraphSelector, onHit go2x3.OnGraphHit) {
	if sel.Traces != nil {
		if sel.Factor {
			cat.selectFactorizations(&sel, onHit)
		} else {
			cat.selectByTraces(&sel, onHit)
		}
	} else {
		cat.selectEncodings(&sel, onHit)
	}
}

const (
	tracesOfs = 1
	kIsPrime  = byte(0x01)
)

func loadAndPushGraph(item *badger.Item, onHit go2x3.OnGraphHit) error {
	err := item.Value(func(val []byte) error {
		X, err := lib2x3.NewGraphFromDef(val)
		if err != nil {
			return err
		}
		onHit <- X
		return nil
	})
	if err != nil {
		panic(err)
	}
	return err
}

func (cat *catalog) selectEncodings(sel *go2x3.GraphSelector, onHit go2x3.OnGraphHit) {
	minKey := [1]byte{sel.Min.NumVerts}

	txn := cat.db.NewTransaction(false)
	defer txn.Discard()

	it := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   300,
	})
	defer it.Close()

	var keyBuf [256]byte
	tracesKey := append(keyBuf[:0], 0xFF, 0xFF) // ensure no match

	for it.Seek(minKey[:]); it.Valid(); {
		curItem := it.Item()
		curKey := curItem.Key()

		// Stop when the vtx count is over the max
		if curKey[0] > sel.Max.NumVerts {
			break
		}

		nextTraces := false

		if bytes.HasPrefix(curKey, tracesKey) {

			loadAndPushGraph(curItem, onHit)

			if sel.UniqueTraces {
				nextTraces = true
			}
		} else {
			n := len(curKey)
			if curKey[n-2] != 0 || curKey[n-1] != 0 { // check double NUL suffix
				panic("what is this entry?")
			}

			// A new prefix means a new Traces entry
			tracesKey = append(tracesKey[:0], curKey...)
			tracesIsPrime := (curItem.UserMeta() & kIsPrime) != 0

			// If only looking for primes and this Traces isn't one, skip to the next
			if sel.PrimesOnly && !tracesIsPrime {
				nextTraces = true
			}
		}

		// If only looking for primes and this Traces isn't one, skip to the next
		if nextTraces {
			tracesKey[len(tracesKey)-1] = 1
			it.Seek(tracesKey)
		} else {
			it.Next()
		}

	}
}

// Currently, the major downside with the current impl is that to read in all the primes requires a complete walk through the TracesCatalog.
func (cat *catalog) readPrimes(
	txn *badger.Txn,
	Nv byte,
	onHit go2x3.OnGraphHit,
) {
	minKey := [1]byte{Nv}

	it := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: false,
		Prefix:         minKey[:1],
	})
	defer it.Close()

	var keyBuf [256]byte
	tracesKey := append(keyBuf[:0], 0xFF, 0xFF) // ensure no match

	for it.Rewind(); it.Valid(); {
		curItem := it.Item()
		curKey := curItem.Key()

		klen := len(curKey)
		if curKey[klen-2] == 0 && curKey[klen-1] == 0 { // check double NUL suffix

			tracesKey := append(tracesKey[:0], curKey...)

			isPrime := (curItem.UserMeta() & kIsPrime) != 0
			if isPrime {
				// We could also load primes directly from a primes table (also allowing us to easily store the "common" encoding of a prime)
				it.Next()
				loadAndPushGraph(it.Item(), onHit)
			}

			// A new prefix means a new Traces entry (change NUL to 0x01 as a means to go to the next entry)
			tracesKey[klen-1] = 1
			it.Seek(tracesKey)

		} else {
			panic("expected Traces entry")
		}
	}
}

/*
func sliceGraphEncoding(tracesKey []byte) []byte {

	// Skip to the end of the TracesSpec (encoded as two NULs)
	XencOffset := -1
	for i := tracesOfs; i < len(tracesKey)-1; i++ {
		if tracesKey[i] == 0 && tracesKey[i+1] == 0 {
			XencOffset = i + 2
			return tracesKey[XencOffset:]
		}
	}

	panic("didn't find end of TracesLSM")
}

func (cat *catalog) lookupTracesID(txn *badger.Txn, X *lib2x3.Graph, autoAdd bool) (tid go2x3.TracesID, wasAdded bool) {
	var keyBuf [256]byte
	tracesKey := cat.formTracesKey(keyBuf[:0], X)
	item, err := txn.Get(tracesKey)
	if err == badger.ErrKeyNotFound {
		if !autoAdd {
			return
		}

		tid = cat.issueNextTracesID(X.NumVerts())

		// Alloc a scrap buf since we can't use the stack for commit bufs
		trLen := len(tracesKey)
		obuf := make([]byte, trLen + go2x3.TracesIDSz)
		tracesKey = append(obuf[:0], tracesKey...)
		trVal := tid.Marshal(tracesKey[trLen:trLen])

		err = txn.Set(tracesKey, trVal)
		wasAdded = true
	}

	if err != nil {
		panic(err)
	}

	err = item.Value(func(val []byte) error {
		return tid.Unmarshal(val)
	})
	if err != nil {
		panic(err)
	}

	return
}
*/

func (cat *catalog) selectByTraces(sel *go2x3.GraphSelector, onHit go2x3.OnGraphHit) {
	if sel.Traces == nil {
		return
	}

	var keyBuf [256]byte
	tracesKey := cat.formTracesKey(keyBuf[:0], sel.Traces)

	txn := cat.db.NewTransaction(false)
	defer txn.Discard()

	it := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   100,
		Prefix:         tracesKey,
	})
	defer it.Close()

	// First item should be the Traces entry header entry.  If not present, then there are no particles with a matching Traces.
	it.Rewind()
	if !it.Valid() {
		return
	}

	// Diagnostic -- the first key we match should be the Traces only key
	{
		curKey := it.Item().Key()

		klen := len(curKey)
		if curKey[klen-2] != 0 || curKey[klen-1] != 0 { // check double NUL suffix
			panic("expected Traces header entry")
		}
	}

	//uidOfs := len(tracesKey)

	// Step over the Traces header entry and read each GraphEncoding
	for it.Next(); it.Valid(); it.Next() {
		//curKey := it.Item().Key()

		/*
			// Skip to the end of the TracesSpec (encoded as two NULs)
			uidOfs := -1
			for i := len(tracesKey); i < len(curKey)-1; i++ {
				if curKey[i] == 0 && curKey[i+1] == 0 {
					encOfs = i + 2
					break
				}
			}

			if uidOfs < 0 {
				panic("end of traces key not found")
			} */

		loadAndPushGraph(it.Item(), onHit)
	}
}

/*
func (cat *catalog) selectByTracesID(tid go2x3.TracesID, onHit lib2x3.OnGraphHit) {
	if tid == 0 {
		return
	}

	var buf [encOfs]byte
	buf[0] = kEncodingCatalog
	prefix := tid.Marshal(buf[:1])

	txnRO := cat.db.NewTransaction(false)
	defer txnRO.Discard()

	it := txnRO.NewIterator(badger.IteratorOptions{
		PrefetchValues: false,
		Prefix:         prefix,
	})
	defer it.Close()

	// Iterate for all encodings having the given TracesID
	for it.Rewind(); it.Valid(); it.Next() {
		curKey := it.Item().Key()

		// First entry should be the Traces value
		if len(curKey) == encOfs {
			continue
		}

		Xenc := lib2x3.GraphEncoding(curKey[encOfs:])
		onHit(tid, Xenc)
	}
}



func (cat *catalog) getTracesIDFromTraces(seeker *easySeeker, TX Traces) TracesID {
	var tracesKeyBuf [256]byte
	tracesKey := cat.formTracesKey(tracesKeyBuf[:0], TX, false)

	tid := TracesID(0)
	seeker.SeekAndGet(tracesKey, func(val []byte) error {
		tid = ReadTracesID(val)
		return nil
	})
	return tid
}

func (cat *catalog) getTracesID(txn *badger.Txn, tracesKey []byte, numVerts byte) (tid TracesID, wasAdded bool) {

	item, err := txn.Get(tracesKey) // Check kTracesCatalog
	if err == nil {
		err = item.Value(func(val []byte) error {
			tid = ReadTracesID(val)
			return nil
		})
	} else if err == badger.ErrKeyNotFound {

		// Create a new TID
		tid = cat.issueNextTracesID(numVerts)
		var tidBuf [8]byte
		tidKey := tid.CatalogKey(tidBuf[:0])

		err = txn.Set(tracesKey, tidKey[1:]) // Post to kTracesCatalog
		if err == nil {
			err = txn.Set(tidKey, tracesKey[1:]) // Post to kTIDCatalog
			wasAdded = true
		}
	}

	if err != nil {
		panic(err)
	}
	return
}

func hasTraces(txn *badger.Txn, tracesKey []byte) bool {
	it := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: false,
		Prefix:         tracesKey,
	})
	defer it.Close()

	it.Rewind()
	return it.Valid()
}

func (cat *catalog) lookupGraph(X *lib2x3.Graph) (isNewTraces, isNewGraph bool) {
	txn := cat.db.NewTransaction(false)
	defer txn.Discard()

	var keyBuf [256]byte
	tracesKey := cat.formTracesKey(keyBuf[:0], X)

	item, err := txn.Get(tracesKey)
	if err == badger.ErrKeyNotFound {
		if !autoAdd {
			return
		}

		tid = cat.issueNextTracesID(X.NumVerts())

		// Alloc a scrap buf since we can't use the stack for commit bufs
		trLen := len(tracesKey)
		obuf := make([]byte, trLen + go2x3.TracesIDSz)
		tracesKey = append(obuf[:0], tracesKey...)
		trVal := tid.Marshal(tracesKey[trLen:trLen])

		err = txn.Set(tracesKey, trVal)
		wasAdded = true
	}

	if err != nil {
		panic(err)
	}


	tid, isNewTraces := cat.lookupTracesID(txn, X, true)

	var buf [256]byte
	buf[0] = kEncodingCatalog
	tidKey := tid.Marshal(buf[:1])
	encKey := X.AppendGraphEncodingTo(tidKey)

	var err error
	if isNewTraces {
		isNewGraph = true
	} else {
		_, err = txn.Get(encKey)
		if err == badger.ErrKeyNotFound {
			isNewGraph = true
		}
	}

	if isNewGraph {
		err = txn.Set(encKey, nil)
	}

	if err == nil {
		err = txn.Commit()
	}

	if err != nil {
		panic(err)
	}

	return
}
*/

// TryAddGraph add the given particle if it doesn't already exist (in its current form).
//
// If true is returned, X was not present and was added.
//
// If false is returned, X already exists in the particle registry (or the graph is not valid
func (cat *catalog) TryAddGraph(X go2x3.GraphState) bool {
	var keyBuf [256]byte
	tracesKey := cat.formTracesKey(keyBuf[:0], X)
	completeKey, err := X.ExportStateEncoding(tracesKey, go2x3.ExportGraphState)
	if err != nil {
		return false
	}
	isNewTraces := false
	isNewGraph := false

	// First see if we have this graph
	txn := cat.db.NewTransaction(true)
	defer txn.Discard()

	_, err = txn.Get(tracesKey)
	if err == badger.ErrKeyNotFound {
		isNewTraces = true
		isNewGraph = true
	} else {
		_, err = txn.Get(completeKey)
		if err == badger.ErrKeyNotFound {
			isNewGraph = true
		}
	}

	// If nothing new, we're done
	if isNewTraces {
		cat.issueNextTracesID(X.NumVertices())
	} else if !isNewGraph {
		return false
	}

	tracesFlags := byte(0)

	// If this Traces hasn't been prime tested before and this is a prime catalog, then do so now.
	if isNewTraces && cat.state.IsPrimeCatalog {

		// Importantly, prime testing only requires primes up to Nv-1
		Nv := X.NumVertices()
		cat.cachePrimesAsNeeded(int(Nv - 1))

		TX := X.Traces(0)
		if cat.primeCache.IsPrime(TX) {
			tracesFlags |= kIsPrime
			cat.issueNextPrimeID(Nv)
		}
	}

	// Write the new entries
	{
		if isNewTraces {
			txn.SetEntry(badger.NewEntry(tracesKey, nil).WithMeta(tracesFlags))
		}
		if isNewGraph {
			var encBuf [128]byte
			val, err := X.ExportStateEncoding(encBuf[:0], go2x3.ExportGraphDef)
			if err == nil {
				txn.Set(completeKey, val)
			}
		}

		err = txn.Commit()
		if err != nil {
			panic(err)
		}
	}

	return isNewGraph
}

// TODO: move to factor.go, i.e. factorCatalog.SelectFactorizations(cat, sel, onHit)
func (cat *catalog) selectFactorizations(sel *go2x3.GraphSelector, onHit go2x3.OnGraphHit) {
	if sel.Traces == nil {
		return
	}

	TX := sel.Traces.Traces(0)

	Nv := len(TX)
	cat.cachePrimesAsNeeded(Nv)

	factorSetsIn := cat.primeCache.FindFactorizations(TX)

	// With all factorizations in hand, we can now iterate and know we have unique instances (since we sorted canonically)
	{
		txnRO := cat.db.NewTransaction(false)
		defer txnRO.Discard()

		seeker := newEasySeeker(txnRO)
		defer seeker.Close()

		for factorSet := range factorSetsIn {
			X := cat.formGraphFromFactors(seeker, factorSet)
			onHit <- X
		}
	}
}

// func (cat *catalog) visitAllTraces(
// 	txn *badger.Txn,
// 	maxNumVerts byte,
// 	maxNumTraces byte,
// 	onTraces func(tid TracesID, Ti Traces) bool) {

// 	itr := txn.NewIterator(badger.IteratorOptions{
// 		PrefetchValues: true,
// 		PrefetchSize:   1000,
// 		Prefix:         byTIDCatalog[:],
// 	})
// 	defer itr.Close()

// 	var tracesBuf [MaxVtxID]int64
// 	Ti := Traces(tracesBuf[:0])

// 	// Loop through all the primes smaller in vertex size than TX
// 	for itr.Rewind(); itr.Valid(); itr.Next() {
// 		item := itr.Item()

// 		// The TID catalog is both a Traces and factor listing, and we only want the Traces.
// 		key := item.Key()[1:]
// 		if len(key) != TracesIDSz {
// 			continue
// 		}
// 		tid := ReadTracesID(key)

// 		// We're done when we hit the vertex limit
// 		if tid.NumVerts() > maxNumVerts {
// 			break
// 		}
// 		err := item.Value(func(val []byte) error {
// 			return Ti.InitFromTracesLSM(val, int(maxNumTraces))
// 		})
// 		if err != nil {
// 			panic(err)
// 		}

// 		if !onTraces(tid, Ti) {
// 			break
// 		}
// 	}
// }

/*
func (cat *catalog) calculateAllFactors(
	TX Traces,
	onFactor func(factor, remainder TracesID),
) {

	txnRO := cat.db.NewTransaction(false)
	defer txnRO.Discard()

	itr := txnRO.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   3000,
		Prefix:         byTIDCatalog[:],
	})
	defer itr.Close()

	seeker := newEasySeeker(txnRO)
	defer seeker.Close()

	var tracesBuf [MaxVtxID]int64
	Ti := Traces(tracesBuf[:0])

	numVerts := byte(len(TX))

	// Loop through all the primes smaller in vertex size than TX
	for itr.Rewind(); itr.Valid(); itr.Next() {
		item := itr.Item()

		// The TID catalog is both a Traces and factor listing, and we only want the Traces.
		key := item.Key()[1:]
		if len(key) != TracesIDSz {
			continue
		}
		factorTID := ReadTracesID(key)

		// We can only consider factors of a smaller vertex size than TX
		if factorTID.NumVerts() >= numVerts {
			break
		}
		err := item.Value(func(val []byte) error {
			return Ti.InitFromTracesLSM(val, int(numVerts))
		})
		if err != nil {
			panic(err)
		}

		// Form the traces key corresponding to (TX - TXi)
		TX.Subtract(Ti, &Ti)

		// If the remainder is not found, then the prospective factor is not valid
		remainderTID := cat.getTracesIDFromTraces(&seeker, Ti)
		if remainderTID != 0 {
			onFactor(factorTID, remainderTID)
		}
	}
}


func (cat *catalog) factorsForTID(
	txn *badger.Txn,
	tid TracesID,
	onFactor func(factor, remainder TracesID),
) {
	var keyBuf [16]byte

	// Prefix such that only the requested TracesID is visited
	tidKey := tid.CatalogKey(keyBuf[:0])
	itr := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: false,
		Prefix:         tidKey,
	})
	defer itr.Close()

	for itr.Rewind(); itr.Valid(); itr.Next() {

		// Skip TID => Traces entries (this should always be the first entry)
		key := itr.Item().Key()[1:]
		if len(key) <= TracesIDSz {
			continue
		}

		factorTID := ReadTracesID(key[4:8])
		remainTID := ReadTracesID(key[8:12])
		onFactor(factorTID, remainTID)
	}
}
*/

func (cat *catalog) cachePrimesAsNeeded(Nv int) error {

	if cat.primeCache == nil {
		return errors.New("not a prime catalog, son")
	}

	have_vi := cat.primeCache.HasFactorsUpTo()
	if have_vi >= Nv {
		return nil
	}

	txnRO := cat.db.NewTransaction(false)
	defer txnRO.Discard()

	numTraces := cat.TraceCount()

	for vi := have_vi + 1; vi <= Nv; vi++ {
		cat.primeCache.NumFactorsHint(vi, cat.state.NumPrimes[vi])

		// Used a buffered channel so that db I/O blocks don't stall Traces computation
		onPrime := make(chan go2x3.GraphState, 4)

		go func() {
			cat.readPrimes(txnRO, byte(vi), onPrime)
			close(onPrime)
		}()

		for Xpr := range onPrime {
			TX := Xpr.Traces(numTraces)
			cat.primeCache.AddCopy(byte(vi), TX)
			Xpr.Reclaim()
		}
	}

	return nil
}

/*
func (cat *catalog) readPrimes(
	txn *badger.Txn,
	v_lo, v_hi byte,
	onHit func(tid go2x3.TracesID, tracesKey []byte),
) {
	itr := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: true,
		PrefetchSize:   355,
		Prefix:         primesPrefix[:],
	})
	defer itr.Close()

	var keyBuf [16]byte
	keyBuf[0] = kPrimeCatalog
	keyBuf[1] = v_lo

	var tracesBuf [256]byte
	tracesBuf[0] = kTracesCatalog
	tracesKey := tracesBuf[:1]

	for itr.Seek(keyBuf[:2]); itr.Valid(); itr.Next() {
		item := itr.Item()

		key := item.Key()[1:]
		if len(key) != TracesIDSz {
			panic("unexpected key size")
		}
		primeTID := ReadTracesID(key)

		// Only consider primes from the range we're given
		if primeTID.NumVerts() > v_hi {
			break
		}

		err := item.Value(func(val []byte) error {
			tracesKey = append(tracesKey[:1], val...)
			tracesKey = append(tracesKey, 0, 0)
			return nil
		})
		if err != nil {
			panic(err)
		}

		onHit(primeTID, tracesKey)
	}
}
*/

// func (cat *catalog) fetch( ) GraphEncoding {

// }

func (cat *catalog) formGraphFromFactors(
	seeker easySeeker,
	primeFactors go2x3.FactorSet,
) *lib2x3.Graph {

	X := lib2x3.NewGraph(nil)
	Xi := lib2x3.NewGraph(nil)

	var keyBuf [256]byte
	for _, Pi := range primeFactors {
		tracesKey := cat.formCatalogKeyFromPrimeFactor(keyBuf[:0], Pi.ID)

		err := seeker.SeekAndGetFirstSub(tracesKey, func(val []byte) error {
			err := Xi.InitFromDef(val)
			return err
		})
		if err == nil {
			for fi := uint32(0); fi < Pi.Count; fi++ {
				X.Concatenate(Xi)
			}
		} else {
			panic(err)
		}
	}

	Xi.Reclaim()
	return X
}

/*
func (cat *catalog) selectPrimes(sel lib2x3.GraphSelector, onHit func(primeTID go2x3.TracesID, Xenc lib2x3.GraphEncoding)) {
	txnRO := cat.db.NewTransaction(false)
	defer txnRO.Discard()

	encReader := newEasySeeker(txnRO)
	defer encReader.Close()

	cat.readPrimes(txnRO, sel.Min.NumVerts, sel.Max.NumVerts, func(primeTID go2x3.TracesID, tracesKey []byte) {
		Xenc, found := encReader.SeekPrefix(tracesKey, tracesKey[len(tracesKey):])
		if found {
			onHit(primeTID, Xenc)
		} else {
			panic("prime not found")
		}
	})
}
*/

type easySeeker struct {
	*badger.Iterator
}

func newEasySeeker(txn *badger.Txn) easySeeker {
	itr := txn.NewIterator(badger.IteratorOptions{
		PrefetchValues: false,
	})
	return easySeeker{itr}
}

func (seeker easySeeker) SeekAndGet(prefix []byte, getter func(val []byte) error) error {
	seeker.Seek(prefix)
	if seeker.Valid() {
		item := seeker.Item()
		if bytes.HasPrefix(item.Key(), prefix) {
			return item.Value(getter)
		}
	}
	return badger.ErrKeyNotFound
}

func (seeker easySeeker) SeekAndGetFirstSub(prefix []byte, getter func(val []byte) error) error {
	seeker.Seek(append(prefix, 0)) // append a NUL to get the entry *after* the prefix entry
	if seeker.Valid() {
		item := seeker.Item()
		if bytes.HasPrefix(item.Key(), prefix) {
			return item.Value(getter)
		}
	}
	return badger.ErrKeyNotFound
}

func (seeker easySeeker) SeekPrefix(prefix []byte, outSuffix []byte) ([]byte, bool) {
	seeker.Seek(prefix)
	if seeker.Valid() {
		key := seeker.Item().Key()
		if bytes.HasPrefix(key, prefix) {
			prefixLen := len(prefix)
			return append(outSuffix, key[prefixLen:]...), true
		}
	}
	return nil, false
}

func (seeker easySeeker) SeekFirstSub(prefix []byte, outSuffix []byte) ([]byte, bool) {
	seeker.Seek(append(prefix, 0)) // append a NUL to get the entry *after* the prefix entry
	if seeker.Valid() {
		key := seeker.Item().Key()
		if bytes.HasPrefix(key, prefix) {
			prefixLen := len(prefix)
			return append(outSuffix, key[prefixLen:]...), true
		}
	}
	return nil, false
}
