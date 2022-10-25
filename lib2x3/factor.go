package lib2x3

import (
	"sync"
	"unsafe"

	"github.com/emirpasic/gods/trees/redblacktree"
)

// FactorSet specifies for many times each possible prime appears
type FactorSet []FactorRun

type FactorRun struct {
	ID    TracesID
	Count uint32
}

func (factors *FactorSet) Insert(toAdd TracesID) {
	insertAt := len(*factors)

	for i, Fi := range *factors {
		if Fi.ID == toAdd {
			(*factors)[i].Count++
			return
		} else if Fi.ID > toAdd {
			insertAt = i
			break
		}
	}

	fax := append((*factors), FactorRun{})
	N := len(fax)
	copy(fax[insertAt+1:N], fax[insertAt:N-1])
	fax[insertAt] = FactorRun{
		ID:    toAdd,
		Count: 1,
	}
	*factors = fax
}

func FactorSetComparator(A, B FactorSet) int {
	lenB := len(B)

	for i, ai := range A {
		if lenB == i {
			return 1
		}

		bi := B[i]
		dID := int(ai.ID) - int(bi.ID)
		if dID != 0 {
			return dID
		}
		dCount := int(ai.Count) - int(bi.Count)
		if dCount != 0 {
			return dCount
		}
	}

	if len(A) > lenB {
		return -1
	}

	return 0
}

func (factors *FactorSet) Clear() {
	*factors = (*factors)[:0]
}

func (factors FactorSet) TotalVtxCount() byte {
	Nv := byte(0)
	for _, Fi := range factors {
		Nv += byte(Fi.Count) * Fi.ID.NumVerts()
	}
	return Nv
}

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
type Factor struct {
	Traces Traces
	ID     TracesID
}

type FactorTable struct {
	TracesPerFactor uint32
	FactorTraces    []int64
	numFactors      uint32
}

func (ft *FactorTable) AddCopy(factor Traces) {
	ft.FactorTraces = append(ft.FactorTraces, factor...)
	ft.numFactors++
}

func (ft *FactorTable) NumFactors() uint32 {
	return ft.numFactors
}

// dst <= A - Factors[Fi]
// Returns true if diff is all zeros.
func (ft *FactorTable) SubtractFactor(dst Traces, A Traces, Fi uint32) bool {
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

func (fcat *FactorCatalog) GetFactorTraces(Nv, Fi uint32) Traces {
	ft := fcat.forNv[Nv]
	i0 := uint32(Fi) * ft.TracesPerFactor
	return Traces(ft.FactorTraces[i0 : i0+ft.TracesPerFactor])
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

func (fcat *FactorCatalog) AddCopy(Nv byte, factor Traces) {
	fcat.forNv[Nv].AddCopy(factor)
}

var factorSearchPool = sync.Pool{
	New: func() interface{} {
		return new(factorSearch)
	},
}

func (fcat *FactorCatalog) FindFactorizations(TX Traces) <-chan FactorSet {
	s := NewFactorSearch(TX, fcat)

	go func() {
		s.SearchForFactors()
		s.onFactor <- 0 // completion signal
	}()

	factorizations := redblacktree.Tree{
		Comparator: func(A, B interface{}) int {
			A0 := A.(FactorSet)
			B0 := B.(FactorSet)
			return FactorSetComparator(A0, B0)
		},
	}

	// Populate factorizations
	{
		const (
			groundState = 0
			readingRun  = 1
		)

		var factorsBuf [MaxVtxID]FactorRun
		curSet := FactorSet(factorsBuf[:0])

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
					newSet := append(FactorSet{}, curSet...)
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

	outlet := make(chan FactorSet)

	go func() {
		itr := factorizations.Iterator()
		for itr.Next() {
			outlet <- itr.Key().(FactorSet)
		}
		close(outlet)
	}()

	return outlet
}

func (fcat *FactorCatalog) IsPrime(TX Traces) bool {
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
	stack       []FactorStep
	onFactor    chan TracesID
	hitCount    int32
	maxFactorSz int32
	Nv          int32
	fcat        *FactorCatalog
	primeTest   bool
	tracesBuf   Traces
}

func (s *factorSearch) sendFactorization(depth int32) bool {
	s.hitCount++
	if s.primeTest {
		return false
	}

	for i := int32(1); i <= depth; i++ {
		Nv := s.stack[i].Nv
		Fi := s.stack[i].FactorIdx
		s.onFactor <- FormTracesID(byte(Nv), uint64(Fi))
	}
	s.onFactor <- 0 // factorization termination signal
	return true
}

func NewFactorSearch(target Traces, fcat *FactorCatalog) *factorSearch {
	s := factorSearchPool.Get().(*factorSearch)

	Nv := len(target)

	if s.onFactor == nil {
		s.onFactor = make(chan TracesID, 8)
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
			s.tracesBuf = make(Traces, max(stackSz*Nv, 9*8))
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
	Remainder Traces
}
