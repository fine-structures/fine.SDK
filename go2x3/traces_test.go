package go2x3

import (
	"testing"
)

var gT *testing.T

func TestTracesEnc(t *testing.T) {
	gT = t
	T1 := Traces([]int64{10, 123, -1234, -12345, 678910, -8765432311})

	{
		var scrap1 [4]byte
		checkEncoding(T1, scrap1[:])
	}

	{
		var scrap1 [200]byte
		checkEncoding(T1, scrap1[:])
	}
}

func checkEncoding(TX Traces, scrap []byte) {

	enc := TX.AppendTracesLSM(scrap[:0])

	var TXdec Traces
	err := TXdec.InitFromTracesLSM(enc, 0)
	if err != nil {
		gT.Fatalf("Traces encoding error: %v", err)
	}

	if TX.IsEqual(TXdec) == false {
		gT.Fatalf("Traces encoding failed, should be:\n     %v\ngot:\n    %v", TX, TXdec)
	}

}
