package graph

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Traces is an arbitrarily length sequence of a phoneix Graph "Trace" values.
type Traces []int64

// TracesRank is a deterministic numerical ranking based on Traces (for a constant number of elements -- typically 12+)
// TracesRank:
//
//	(a) serves as a hash for Traces, enhancing db query performance, and
//	(b) orders a catalog's kTracesCatalog in a way pleasing to even the pickiest physicist (ok fine, that's physically impossible).
type TracesRank uint64

// TracesLSM is a LSM binary encoding / symbol of a Traces.
type TracesLSM []byte

// TracesID uniquely identifies a Graph's trace spectrum ("Traces")
type TracesID uint64

// // TODO: attach to graph struct
// type SysTraces struct {

// 	//Terms []*TracesTerm
// }

// func (x *SysTraces) Clear() {

// }

// func (x *SysTraces) AssignFromTraces(TX Traces)  {

// }

// func (x *SysTraces) MakeCanonic() {

// 	sort.Slice(x.Terms, func(i, j int) bool {
// 		return false	// TODO
// 	})
// }

// func (sys *SysTraces) AppendLookupKey([]byte) []byte {
// 	return nil // TODO
// }

func mini(a, b int) int {
	if a < b {
		return a
	} else {
		return b
	}
}

// IsEqual returns if two traces have the same prefix.
// The number of elements compared is the trace with the shorter length, so a Traces of length 0 will be equal to all other Traces.
func (TX Traces) IsEqual(target Traces) bool {
	N := mini(len(TX), len(target))
	for i := 0; i < N; i++ {
		if TX[i] != target[i] {
			return false
		}
	}
	return true
}

// IsZero() returns true if all values of this Traces are 0.
func (TX Traces) IsZero() bool {
	for _, TXi := range TX {
		if TXi != 0 {
			return false
		}
	}
	return true
}

// Computes TX-delta and places the result into diff.
// Returns true of the result is all zeros.
func (TX Traces) Subtract(delta Traces, diff *Traces) (isZero bool) {
	N := mini(len(TX), len(delta))

	isZero = true
	diff.SetLen(N)
	for i := 0; i < N; i++ {
		di := TX[i] - delta[i]
		(*diff)[i] = di
		if di != 0 {
			isZero = false
		}
	}
	return isZero
}

// CalcTracesRank calculates the TracesRank for this Traces (and assumes Nv == len(TX))
// TracesRank:
//
//	(a) serves as a hash for Traces, enhancing db query performance, and
//	(b) orders a catalog's kTracesCatalog in a way pleasing to even the pickiest physicist (ok fine, that's physically impossible).
func (TX Traces) CalcTracesRank() TracesRank {
	return 0
}

func (TX *Traces) SetLen(tracesLen int) {
	if cap(*TX) < tracesLen {
		dimLen := tracesLen
		if dimLen < 16 {
			dimLen = 16 // prevent rapid resizing
		}
		*TX = make([]int64, tracesLen, dimLen)
	} else {
		*TX = (*TX)[:tracesLen]
	}
}

// InitFromTracesLSM assigns this Traces from a binary encoding made from AppendTracesLSM()
func (TX *Traces) InitFromTracesLSM(Xkey TracesLSM, maxNumTraces int) error {
	out := (*TX)[:0]
	rdr := bytes.NewReader(Xkey)

	var err error
	for {
		trace, err := binary.ReadVarint(rdr)
		if err != nil {
			if err == io.ErrShortBuffer {
				err = nil
			}
			break
		}
		out = append(out, trace)
		if maxNumTraces > 0 && len(out) >= maxNumTraces {
			break
		}
	}

	*TX = out
	return err
}

// AppendTracesLSM appends a canonical binary encoding of TX to []out, returning it as TracesLSM.
//
// The integer returned a the byte length in the returned TracesLSM after TX.VtxCount exported elements.
func (TX Traces) AppendTracesLSM(out []byte) TracesLSM {
	return TX.appendOddEvenEncoding(out)
}

func (TX Traces) appendOddEvenEncoding(out []byte) TracesLSM {
	numTraces := len(TX)
	var scrap [12]byte

	// Odd traces first -- normalize sign to first non-zero off term
	key := out
	sign := TracesSign_Zero
	for i := 0; i < numTraces; i += 2 {
		Ti := TX[i]
		if sign == TracesSign_Zero {
			if Ti > 0 {
				sign = TracesSign_Pos
			} else if Ti < 0 {
				sign = TracesSign_Neg
			}
		}
		if sign == TracesSign_Neg {
			Ti = -Ti
		}
		n := binary.PutVarint(scrap[:], Ti)
		key = append(key, scrap[:n]...)
	}

	// Even traces
	for i := 1; i < numTraces; i += 2 {
		Ti := TX[i]
		// if oddNorm == 0 {
		// 	if Ti & 1 != 0 {
		// 		panic("even trace on boson is odd")
		// 	}
		// 	Ti = Ti >> 1
		// }
		n := binary.PutVarint(scrap[:], Ti)
		key = append(key, scrap[:n]...)
	}

	// Append sign (last)
	key = append(key, byte(sign))

	return key
}

const TracesIDSz = 7

func FormTracesID(numVerts uint32, seriesID uint64) TracesID {
	return TracesID((uint64(numVerts) << 48) | uint64(seriesID))
}

func (tid TracesID) Marshal(in []byte) []byte {
	return append(in,
		byte(tid>>48),
		byte(tid>>40),
		byte(tid>>32),
		byte(tid>>24),
		byte(tid>>16),
		byte(tid>>8),
		byte(tid),
	)
}

func (tid *TracesID) Unmarshal(in []byte) error {
	if len(in) < TracesIDSz {
		*tid = 0
		return ErrUnmarshal
	}
	*tid = TracesID(
		(uint64(in[0]) << 48) | // MSB is the vertex count
			(uint64(in[1]) << 40) |
			(uint64(in[2]) << 32) |
			(uint64(in[3]) << 24) |
			(uint64(in[4]) << 16) |
			(uint64(in[5]) << 8) |
			(uint64(in[6])))
	return nil
}

func (tid TracesID) NumVertices() uint32 {
	return uint32(byte(tid >> 48))
}

func (tid TracesID) SeriesID() uint64 {
	return 0xFFFFFFFFFFFF & uint64(tid)
}

func (tid TracesID) WriteAsString(out io.Writer) {
	fmt.Fprintf(out, "%d-%d", tid.NumVertices(), tid.SeriesID())
}
