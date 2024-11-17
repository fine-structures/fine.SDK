package factor

import (
	"sync"
	"unsafe"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/fine-structures/fine.SDK/go2x3"
)

/*
func (factors FactorSet) AppendEncodingTo(out []byte) []byte {
	var scrap [binary.MaxVarintLen64]byte

	// Leads with the number of factors and then write each factor (but we won't know until the end)
	startIdx := len(out)
	out = append(out, 0)
	numUniques := byte(0)

	for _, Fi := range factors {
		if Fi.Count > 0 {
			numUniques++
			out = append(out, Fi.Count, Fi.NumVerts)
			n := binary.PutUvarint(scrap[:], uint64(Fi.SeriesID))
			out = append(out, scrap[:n]...)
		}
	}
	out[startIdx] = numUniques

	return out
}

func (factors *PrimeFactors) InitFromEncoding(in []byte) error {
	factorCount := in[0]
	idx := 1

	var facs PrimeFactors
	if int(factorCount) > cap(*factors) {
		facs = make(PrimeFactors, factorCount)
	} else {
		facs = (*factors)[:factorCount]
	}

	for i := byte(0); i < factorCount; i++ {
		ID, n := binary.Uvarint(in[idx+2:])

		facs[i] = Prime{
			Count:    in[idx],
			NumVerts: in[idx+1],
			SeriesID: uint32(ID),
		}
		idx += n + 2
	}

	if idx != len(in) {
		return io.EOF
	}

	*factors = facs
	return nil
}
*/

// FactorTable is a set of factors each having the same vertex size
type FactorTable struct {
	TracesPerFactor uint32
	FactorTraces    []int64
	Nv              uint32
	numFactors      uint32
}

func (ft *FactorTable) AddCopy(factor go2x3.Traces) {
	ft.FactorTraces = append(ft.FactorTraces, factor...)
	ft.numFactors++
}

func (ft *FactorTable) NumFactors() uint32 {
	return ft.numFactors
}

// Subtracts the Fi'th factor from A, storing the result in dst:
//
//	dst <- A - Factors[Fi]
//
// Returns true if diff is all zeros.
func (ft *FactorTable) SubtractFactor(dst go2x3.Traces, A go2x3.Traces, Fi uint32) bool {
	numTraces := ft.TracesPerFactor
	B := unsafe.Slice(&ft.FactorTraces[Fi*numTraces], numTraces)
	isZero := true
	for i, Ai := range A {
		diff := Ai - B[i]
		dst[i] = diff
		if diff != 0 {
			isZero = false
		}
	}
	return isZero
}

type FactorCatalog struct {
	forNv []*FactorTable
	maxNv int32
}

func (fcat *FactorCatalog) GetFactorTraces(Nv, Fi uint32) go2x3.Traces {
	ft := fcat.forNv[Nv]
	i0 := uint32(Fi) * ft.TracesPerFactor
	return go2x3.Traces(ft.FactorTraces[i0 : i0+ft.TracesPerFactor])
}

func (fcat *FactorCatalog) HasFactorsUpTo() int {
	Nv := 0
	for vi := 1; vi < len(fcat.forNv); vi++ {
		if fcat.forNv[vi].numFactors == 0 {
			break
		}
		Nv = vi
	}
	return Nv
}

func NewFactorCatalog(maxNv int32) *FactorCatalog {
	fcat := &FactorCatalog{
		maxNv: maxNv,
		forNv: make([]*FactorTable, maxNv+1), // one based indexing
	}
	for vi := range fcat.forNv {
		fcat.forNv[vi] = &FactorTable{
			Nv:              uint32(vi),
			TracesPerFactor: uint32(maxNv),
		}
	}
	return fcat
}

func (fcat *FactorCatalog) GetFactorTable(Nv int32) *FactorTable {
	return fcat.forNv[Nv]
}

func (fcat *FactorCatalog) NumFactorsHint(Nv int, numFactors uint64) {
	ft := fcat.forNv[Nv]

	needed := int(numFactors) * int(ft.TracesPerFactor)
	if needed > cap(ft.FactorTraces) {
		oldTraces := ft.FactorTraces
		ft.FactorTraces = make([]int64, len(oldTraces), needed)
		copy(ft.FactorTraces, oldTraces)
	}
}

func (fcat *FactorCatalog) AddCopy(Nv byte, factor go2x3.Traces) {
	fcat.forNv[Nv].AddCopy(factor)
}

var factorSearchPool = sync.Pool{
	New: func() interface{} {
		return new(factorSearch)
	},
}

func (fcat *FactorCatalog) FindFactorizations(TX go2x3.Traces) <-chan go2x3.FactorSet {
	s := NewFactorSearch(TX, fcat)

	go func() {
		s.SearchForFactors()
		s.onFactor <- 0 // completion signal
	}()

	factorizations := redblacktree.Tree{
		Comparator: func(A, B interface{}) int {
			A0 := A.(go2x3.FactorSet)
			B0 := B.(go2x3.FactorSet)
			return go2x3.FactorSetComparator(A0, B0)
		},
	}

	// Populate factorizations
	{
		const (
			groundState = 0
			readingRun  = 1
		)

		var factorsBuf [go2x3.MaxVtxID]go2x3.FactorRun
		curSet := go2x3.FactorSet(factorsBuf[:0])

		state := groundState

		for factor_i := range s.onFactor {

			// A nil prime ID signals the search is complete
			if factor_i == 0 {
				if state == groundState {
					break
				}

				// If the given (canonical) factor set is not yet added, do so
				_, found := factorizations.Get(curSet)
				if !found {
					newSet := append(go2x3.FactorSet{}, curSet...)
					factorizations.Put(newSet, nil)
				}

				state = groundState
				curSet.Clear()
				continue
			}

			state = readingRun
			curSet.Insert(factor_i)
		}
	}

	// Once we have have all possible factorizations, they are stored in the factorizations tree.
	// We can now iterate through our canonical tree and therefore eliminate dupes.
	s.Reclaim()

	outlet := make(chan go2x3.FactorSet)

	go func() {
		itr := factorizations.Iterator()
		for itr.Next() {
			outlet <- itr.Key().(go2x3.FactorSet)
		}
		close(outlet)
	}()

	return outlet
}

func (fcat *FactorCatalog) IsPrime(TX go2x3.Traces) bool {
	if len(TX) <= 1 {
		return true
	}

	s := NewFactorSearch(TX, fcat)
	s.primeTest = true
	s.SearchForFactors()

	isPrime := s.hitCount == 0
	s.Reclaim()

	return isPrime
}

func (s *factorSearch) SearchForFactors() {
	if s.primeTest {
		s.maxFactorSz = s.Nv - 1
	}
	s.findFactors(0, 1, s.Nv)
}

func (s *factorSearch) findFactors(depth, vi_start, Nv_remain int32) bool {
	R0 := s.stack[depth].Remainder

	depth++
	R1 := s.stack[depth].Remainder
	F1 := &s.stack[depth].FactorIdx

	more := true
	for vi := vi_start; vi <= Nv_remain && more; vi++ {
		if vi <= s.maxFactorSz {
			s.stack[depth].Nv = vi
			ft := s.fcat.GetFactorTable(vi)
			numFactors := ft.NumFactors()
			for Fi := uint32(0); Fi < numFactors; Fi++ {
				isZero := ft.SubtractFactor(R1, R0, Fi)
				*F1 = Fi
				if vi < Nv_remain {
					more = s.findFactors(depth, vi, Nv_remain-vi)
				} else if isZero {
					more = s.sendFactorization(depth)
				}
			}
		}
	}

	return more
}

type factorSearch struct {
	stack       []FactorStep        // stack of factorization -- "dynamic programming"
	onFactor    chan go2x3.TracesID // factorizations channel
	hitCount    int32               // number of factorizations found
	maxFactorSz int32               // maximum factor size to consider
	Nv          int32               // number of vertices in the target graph
	fcat        *FactorCatalog      // where prime factors are stored
	primeTest   bool                // if set, we are testing if a graph is prime
	tracesBuf   go2x3.Traces        // scrap buffer
}

func (s *factorSearch) sendFactorization(depth int32) bool {
	s.hitCount++
	if s.primeTest {
		return false
	}

	for i := int32(1); i <= depth; i++ {
		Nv := s.stack[i].Nv
		Fi := s.stack[i].FactorIdx
		s.onFactor <- go2x3.FormTracesID(uint32(Nv), uint64(Fi))
	}
	s.onFactor <- 0 // factorization termination signal
	return true
}

func NewFactorSearch(target go2x3.Traces, fcat *FactorCatalog) *factorSearch {
	s := factorSearchPool.Get().(*factorSearch)

	Nv := len(target)

	if s.onFactor == nil {
		s.onFactor = make(chan go2x3.TracesID, 8)
	}
	s.primeTest = false
	s.hitCount = 0
	s.fcat = fcat
	s.Nv = int32(Nv)
	s.maxFactorSz = s.Nv

	// Use a single large buffer for all the stack traces
	{
		stackSz := Nv + 1
		if cap(s.stack) < stackSz {
			s.stack = make([]FactorStep, max(stackSz, 9))
		}

		tracesSz := stackSz * Nv
		if cap(s.tracesBuf) < tracesSz {
			s.tracesBuf = make(go2x3.Traces, max(stackSz*Nv, 9*8))
		}

		for i := 0; i <= Nv; i++ {
			start := i * Nv
			s.stack[i].Remainder = s.tracesBuf[start : start+Nv]
		}

		copy(s.stack[0].Remainder, target)
		s.stack[0].Nv = 0
	}

	return s
}

func (s *factorSearch) Reclaim() {
	s.fcat = nil
	factorSearchPool.Put(s)
}

type FactorStep struct {
	Nv        int32
	FactorIdx uint32
	Remainder go2x3.Traces
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
