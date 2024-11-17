// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: fine/go2x3.proto

package go2x3

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
	reflect "reflect"
	strings "strings"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type CatalogState struct {
	MajorVers int32 `protobuf:"varint,1,opt,name=MajorVers,proto3" json:"MajorVers,omitempty"`
	MinorVers int32 `protobuf:"varint,2,opt,name=MinorVers,proto3" json:"MinorVers,omitempty"`
	// TraceCount is the Traces len for this Catalog's Traces index.
	// This effectively sets a vertex size limit for Graphs this Catalog can process.
	// DefaultCatalogTraceCount specifies the default TraceCount for new catalogs.
	TraceCount int32 `protobuf:"varint,10,opt,name=TraceCount,proto3" json:"TraceCount,omitempty"`
	// NumTraces[Nv] is the number of traces of in this catalog for a given number of vertices.
	// Note: NumTraces[0] is always 0 and len(NumTraces) == TraceCount+1
	NumTraces []uint64 `protobuf:"varint,11,rep,packed,name=NumTraces,proto3" json:"NumTraces,omitempty"`
	// NumPrimes[Nv] is the number of particle primes for a given number of vertices.
	// Note: NumPrimes[0] is always 0 and len(NumPrimes) == TraceCount+1
	NumPrimes []uint64 `protobuf:"varint,12,rep,packed,name=NumPrimes,proto3" json:"NumPrimes,omitempty"`
	// Set if this catalog is to auto-determine if a newly added Graph / Traces are primes.
	IsPrimeCatalog bool `protobuf:"varint,20,opt,name=IsPrimeCatalog,proto3" json:"IsPrimeCatalog,omitempty"`
}

type Bool int32

const (
	Bool_Unspecified Bool = 0
	Bool_Yes     Bool = 1
	Bool_No      Bool = 3
)

func (m *CatalogState) Reset()      { *m = CatalogState{} }
func (*CatalogState) ProtoMessage() {}
func (*CatalogState) Descriptor() ([]byte, []int) {
	return fileDescriptor_5ef11d9c64c217e5, []int{0}
}
func (m *CatalogState) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CatalogState) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CatalogState.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CatalogState) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CatalogState.Merge(m, src)
}
func (m *CatalogState) XXX_Size() int {
	return m.Size()
}
func (m *CatalogState) XXX_DiscardUnknown() {
	xxx_messageInfo_CatalogState.DiscardUnknown(m)
}

var xxx_messageInfo_CatalogState proto.InternalMessageInfo

func (m *CatalogState) GetMajorVers() int32 {
	if m != nil {
		return m.MajorVers
	}
	return 0
}

func (m *CatalogState) GetMinorVers() int32 {
	if m != nil {
		return m.MinorVers
	}
	return 0
}

func (m *CatalogState) GetTraceCount() int32 {
	if m != nil {
		return m.TraceCount
	}
	return 0
}

func (m *CatalogState) GetNumTraces() []uint64 {
	if m != nil {
		return m.NumTraces
	}
	return nil
}

func (m *CatalogState) GetNumPrimes() []uint64 {
	if m != nil {
		return m.NumPrimes
	}
	return nil
}

func (m *CatalogState) GetIsPrimeCatalog() bool {
	if m != nil {
		return m.IsPrimeCatalog
	}
	return false
}

func init() {
	proto.RegisterType((*CatalogState)(nil), "go2x3.CatalogState")
}

func init() { proto.RegisterFile("fine/go2x3.proto", fileDescriptor_5ef11d9c64c217e5) }

var fileDescriptor_5ef11d9c64c217e5 = []byte{
	// 224 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4f, 0xcb, 0xcc, 0x4b,
	0xd5, 0x07, 0x11, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0x2c, 0x20, 0xb6, 0xd2, 0x39, 0x46,
	0x2e, 0x1e, 0xe7, 0xc4, 0x92, 0xc4, 0x9c, 0xfc, 0xf4, 0xe0, 0x92, 0xc4, 0x92, 0x54, 0x21, 0x19,
	0x2e, 0x4e, 0xdf, 0xc4, 0xac, 0xfc, 0xa2, 0xb0, 0xd4, 0xa2, 0x62, 0x09, 0x46, 0x05, 0x46, 0x0d,
	0xd6, 0x20, 0x84, 0x00, 0x58, 0x36, 0x33, 0x0f, 0x2a, 0xcb, 0x04, 0x95, 0x85, 0x09, 0x08, 0xc9,
	0x71, 0x71, 0x85, 0x14, 0x25, 0x26, 0xa7, 0x3a, 0xe7, 0x97, 0xe6, 0x95, 0x48, 0x70, 0x81, 0xa5,
	0x91, 0x44, 0x40, 0xba, 0xfd, 0x4a, 0x73, 0xc1, 0x02, 0xc5, 0x12, 0xdc, 0x0a, 0xcc, 0x1a, 0x2c,
	0x41, 0x08, 0x01, 0xa8, 0x6c, 0x40, 0x51, 0x66, 0x6e, 0x6a, 0xb1, 0x04, 0x0f, 0x5c, 0x16, 0x22,
	0x20, 0xa4, 0xc6, 0xc5, 0xe7, 0x59, 0x0c, 0x66, 0x43, 0x9d, 0x2b, 0x21, 0xa2, 0xc0, 0xa8, 0xc1,
	0x11, 0x84, 0x26, 0xea, 0x64, 0x72, 0xe1, 0xa1, 0x1c, 0xc3, 0x8d, 0x87, 0x72, 0x0c, 0x1f, 0x1e,
	0xca, 0x31, 0x36, 0x3c, 0x92, 0x63, 0x5c, 0xf1, 0x48, 0x8e, 0xf1, 0xc4, 0x23, 0x39, 0xc6, 0x0b,
	0x8f, 0xe4, 0x18, 0x1f, 0x3c, 0x92, 0x63, 0x7c, 0xf1, 0x48, 0x8e, 0xe1, 0xc3, 0x23, 0x39, 0xc6,
	0x09, 0x8f, 0xe5, 0x18, 0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21, 0x89, 0x0d, 0x1c,
	0x26, 0xc6, 0x80, 0x00, 0x00, 0x00, 0xff, 0xff, 0xf6, 0xdd, 0x51, 0x21, 0x26, 0x01, 0x00, 0x00,
}

func (this *CatalogState) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*CatalogState)
	if !ok {
		that2, ok := that.(CatalogState)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.MajorVers != that1.MajorVers {
		return false
	}
	if this.MinorVers != that1.MinorVers {
		return false
	}
	if this.TraceCount != that1.TraceCount {
		return false
	}
	if len(this.NumTraces) != len(that1.NumTraces) {
		return false
	}
	for i := range this.NumTraces {
		if this.NumTraces[i] != that1.NumTraces[i] {
			return false
		}
	}
	if len(this.NumPrimes) != len(that1.NumPrimes) {
		return false
	}
	for i := range this.NumPrimes {
		if this.NumPrimes[i] != that1.NumPrimes[i] {
			return false
		}
	}
	if this.IsPrimeCatalog != that1.IsPrimeCatalog {
		return false
	}
	return true
}
func (this *CatalogState) GoString() string {
	if this == nil {
		return "nil"
	}
	s := make([]string, 0, 10)
	s = append(s, "&go2x3.CatalogState{")
	s = append(s, "MajorVers: "+fmt.Sprintf("%#v", this.MajorVers)+",\n")
	s = append(s, "MinorVers: "+fmt.Sprintf("%#v", this.MinorVers)+",\n")
	s = append(s, "TraceCount: "+fmt.Sprintf("%#v", this.TraceCount)+",\n")
	s = append(s, "NumTraces: "+fmt.Sprintf("%#v", this.NumTraces)+",\n")
	s = append(s, "NumPrimes: "+fmt.Sprintf("%#v", this.NumPrimes)+",\n")
	s = append(s, "IsPrimeCatalog: "+fmt.Sprintf("%#v", this.IsPrimeCatalog)+",\n")
	s = append(s, "}")
	return strings.Join(s, "")
}
func valueToGoStringFine(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func (m *CatalogState) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CatalogState) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *CatalogState) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.IsPrimeCatalog {
		i--
		if m.IsPrimeCatalog {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0xa0
	}
	if len(m.NumPrimes) > 0 {
		dAtA2 := make([]byte, len(m.NumPrimes)*10)
		var j1 int
		for _, num := range m.NumPrimes {
			for num >= 1<<7 {
				dAtA2[j1] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j1++
			}
			dAtA2[j1] = uint8(num)
			j1++
		}
		i -= j1
		copy(dAtA[i:], dAtA2[:j1])
		i = encodeVarintFine(dAtA, i, uint64(j1))
		i--
		dAtA[i] = 0x62
	}
	if len(m.NumTraces) > 0 {
		dAtA4 := make([]byte, len(m.NumTraces)*10)
		var j3 int
		for _, num := range m.NumTraces {
			for num >= 1<<7 {
				dAtA4[j3] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j3++
			}
			dAtA4[j3] = uint8(num)
			j3++
		}
		i -= j3
		copy(dAtA[i:], dAtA4[:j3])
		i = encodeVarintFine(dAtA, i, uint64(j3))
		i--
		dAtA[i] = 0x5a
	}
	if m.TraceCount != 0 {
		i = encodeVarintFine(dAtA, i, uint64(m.TraceCount))
		i--
		dAtA[i] = 0x50
	}
	if m.MinorVers != 0 {
		i = encodeVarintFine(dAtA, i, uint64(m.MinorVers))
		i--
		dAtA[i] = 0x10
	}
	if m.MajorVers != 0 {
		i = encodeVarintFine(dAtA, i, uint64(m.MajorVers))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintFine(dAtA []byte, offset int, v uint64) int {
	offset -= sovFine(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *CatalogState) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.MajorVers != 0 {
		n += 1 + sovFine(uint64(m.MajorVers))
	}
	if m.MinorVers != 0 {
		n += 1 + sovFine(uint64(m.MinorVers))
	}
	if m.TraceCount != 0 {
		n += 1 + sovFine(uint64(m.TraceCount))
	}
	if len(m.NumTraces) > 0 {
		l = 0
		for _, e := range m.NumTraces {
			l += sovFine(uint64(e))
		}
		n += 1 + sovFine(uint64(l)) + l
	}
	if len(m.NumPrimes) > 0 {
		l = 0
		for _, e := range m.NumPrimes {
			l += sovFine(uint64(e))
		}
		n += 1 + sovFine(uint64(l)) + l
	}
	if m.IsPrimeCatalog {
		n += 3
	}
	return n
}

func sovFine(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozFine(x uint64) (n int) {
	return sovFine(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (this *CatalogState) String() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&CatalogState{`,
		`MajorVers:` + fmt.Sprintf("%v", this.MajorVers) + `,`,
		`MinorVers:` + fmt.Sprintf("%v", this.MinorVers) + `,`,
		`TraceCount:` + fmt.Sprintf("%v", this.TraceCount) + `,`,
		`NumTraces:` + fmt.Sprintf("%v", this.NumTraces) + `,`,
		`NumPrimes:` + fmt.Sprintf("%v", this.NumPrimes) + `,`,
		`IsPrimeCatalog:` + fmt.Sprintf("%v", this.IsPrimeCatalog) + `,`,
		`}`,
	}, "")
	return s
}
func valueToStringFine(v interface{}) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("*%v", pv)
}
func (m *CatalogState) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFine
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: CatalogState: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CatalogState: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MajorVers", wireType)
			}
			m.MajorVers = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFine
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MajorVers |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MinorVers", wireType)
			}
			m.MinorVers = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFine
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MinorVers |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field TraceCount", wireType)
			}
			m.TraceCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFine
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.TraceCount |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 11:
			if wireType == 0 {
				var v uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowFine
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.NumTraces = append(m.NumTraces, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowFine
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthFine
				}
				postIndex := iNdEx + packedLen
				if postIndex < 0 {
					return ErrInvalidLengthFine
				}
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				var elementCount int
				var count int
				for _, integer := range dAtA[iNdEx:postIndex] {
					if integer < 128 {
						count++
					}
				}
				elementCount = count
				if elementCount != 0 && len(m.NumTraces) == 0 {
					m.NumTraces = make([]uint64, 0, elementCount)
				}
				for iNdEx < postIndex {
					var v uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowFine
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.NumTraces = append(m.NumTraces, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field NumTraces", wireType)
			}
		case 12:
			if wireType == 0 {
				var v uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowFine
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.NumPrimes = append(m.NumPrimes, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowFine
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthFine
				}
				postIndex := iNdEx + packedLen
				if postIndex < 0 {
					return ErrInvalidLengthFine
				}
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				var elementCount int
				var count int
				for _, integer := range dAtA[iNdEx:postIndex] {
					if integer < 128 {
						count++
					}
				}
				elementCount = count
				if elementCount != 0 && len(m.NumPrimes) == 0 {
					m.NumPrimes = make([]uint64, 0, elementCount)
				}
				for iNdEx < postIndex {
					var v uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowFine
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.NumPrimes = append(m.NumPrimes, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field NumPrimes", wireType)
			}
		case 20:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IsPrimeCatalog", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFine
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.IsPrimeCatalog = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipFine(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFine
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipFine(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowFine
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFine
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFine
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthFine
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupFine
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthFine
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthFine        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowFine          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupFine = fmt.Errorf("proto: unexpected end of group")
)
