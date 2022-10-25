package lib2x3

import "github.com/dgraph-io/badger/v3"

// CanonicSet allows adding canonical encodings of particle graphs and returning if an equivalent graph has already been added.
type CanonicSet interface {

	// TryAdd adds the given particle graph if it is not already present.
	//
	// If the canonic version of X already is in this CanonicSet, this call has no effect and TryAdd() returns false
	// If X isn't in this set, X is added and TryAdd() returns true .
	//
	// After one or more calls to TryAdd(), call Close() for cleanup.
	TryAdd(X *Graph) bool

	// Close removes all previously added items from this set.
	//
	// If you make subsequent calls to TryAdd(), be sure you call Close() when you're done.
	Close()
}

// TracesSet allows adding of Traces to an internal set and returning if a given Traces has already been added.
type TracesSet interface {

	// TryAdd adds the given Traces if it is not already present.
	//
	// If TX already is in this TracesSet, false is returned and this call has no effect.
	// If TX isn't in this TracesSet, a copy of TX is added and true is returned.
	//
	// After one or more calls to TryAdd(), be sure to call Close() for cleanup.
	TryAdd(TX Traces) bool

	// Close removes all previously added items from this set.
	//
	// If you make subsequent calls to TryAdd(), call Close() when you're done.
	Close()
}

func NewTracesSet() TracesSet {
	return &tracesSet{}
}

func (ts *tracesSet) TryAdd(TX Traces) bool {
	var buf TraceSpecBuf
	key := TX.AppendTraceSpecTo(buf[:0])
	return ts.tryAdd(key)
}

type tracesSet struct {
	lsmSet
}

type lsmSet struct {
	db *badger.DB
}

func (set *lsmSet) autoOpen() {
	if set.db == nil {
		dbOpts := badger.DefaultOptions("").WithInMemory(true)
		dbOpts.Logger = nil
		dbOpts.MetricsEnabled = false

		var err error
		set.db, err = badger.Open(dbOpts)
		if err != nil {
			panic(err)
		}
	}
}

func (set *lsmSet) tryAdd(key []byte) bool {
	set.autoOpen()

	txn := set.db.NewTransaction(true)
	defer txn.Commit()

	added := false
	_, err := txn.Get(key)
	if err == nil {
		// no-op since the key is already in the db
	} else if err == badger.ErrKeyNotFound {
		err = txn.Set(key, nil)
		added = true
	}

	if err != nil {
		panic(err)
	}

	return added
}

func (set *lsmSet) Close() {
	if set.db != nil {
		set.db.Close()
		set.db = nil
	}
}
