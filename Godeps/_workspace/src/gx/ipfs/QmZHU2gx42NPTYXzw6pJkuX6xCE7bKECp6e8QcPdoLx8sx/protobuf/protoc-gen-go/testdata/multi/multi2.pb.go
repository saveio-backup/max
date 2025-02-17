// Code generated by protoc-gen-go. DO NOT EDIT.
// source: multi/multi2.proto

package multitest // import "github.com/golang/protobuf/protoc-gen-go/testdata/multi"

import proto "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZHU2gx42NPTYXzw6pJkuX6xCE7bKECp6e8QcPdoLx8sx/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Multi2_Color int32

const (
	Multi2_BLUE  Multi2_Color = 1
	Multi2_GREEN Multi2_Color = 2
	Multi2_RED   Multi2_Color = 3
)

var Multi2_Color_name = map[int32]string{
	1: "BLUE",
	2: "GREEN",
	3: "RED",
}
var Multi2_Color_value = map[string]int32{
	"BLUE":  1,
	"GREEN": 2,
	"RED":   3,
}

func (x Multi2_Color) Enum() *Multi2_Color {
	p := new(Multi2_Color)
	*p = x
	return p
}
func (x Multi2_Color) String() string {
	return proto.EnumName(Multi2_Color_name, int32(x))
}
func (x *Multi2_Color) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(Multi2_Color_value, data, "Multi2_Color")
	if err != nil {
		return err
	}
	*x = Multi2_Color(value)
	return nil
}
func (Multi2_Color) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_multi2_c47490ad66d93e67, []int{0, 0}
}

type Multi2 struct {
	RequiredValue        *int32        `protobuf:"varint,1,req,name=required_value,json=requiredValue" json:"required_value,omitempty"`
	Color                *Multi2_Color `protobuf:"varint,2,opt,name=color,enum=multitest.Multi2_Color" json:"color,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *Multi2) Reset()         { *m = Multi2{} }
func (m *Multi2) String() string { return proto.CompactTextString(m) }
func (*Multi2) ProtoMessage()    {}
func (*Multi2) Descriptor() ([]byte, []int) {
	return fileDescriptor_multi2_c47490ad66d93e67, []int{0}
}
func (m *Multi2) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Multi2.Unmarshal(m, b)
}
func (m *Multi2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Multi2.Marshal(b, m, deterministic)
}
func (dst *Multi2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Multi2.Merge(dst, src)
}
func (m *Multi2) XXX_Size() int {
	return xxx_messageInfo_Multi2.Size(m)
}
func (m *Multi2) XXX_DiscardUnknown() {
	xxx_messageInfo_Multi2.DiscardUnknown(m)
}

var xxx_messageInfo_Multi2 proto.InternalMessageInfo

func (m *Multi2) GetRequiredValue() int32 {
	if m != nil && m.RequiredValue != nil {
		return *m.RequiredValue
	}
	return 0
}

func (m *Multi2) GetColor() Multi2_Color {
	if m != nil && m.Color != nil {
		return *m.Color
	}
	return Multi2_BLUE
}

func init() {
	proto.RegisterType((*Multi2)(nil), "multitest.Multi2")
	proto.RegisterEnum("multitest.Multi2_Color", Multi2_Color_name, Multi2_Color_value)
}

func init() { proto.RegisterFile("multi/multi2.proto", fileDescriptor_multi2_c47490ad66d93e67) }

var fileDescriptor_multi2_c47490ad66d93e67 = []byte{
	// 202 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xca, 0x2d, 0xcd, 0x29,
	0xc9, 0xd4, 0x07, 0x93, 0x46, 0x7a, 0x05, 0x45, 0xf9, 0x25, 0xf9, 0x42, 0x9c, 0x60, 0x5e, 0x49,
	0x6a, 0x71, 0x89, 0x52, 0x2b, 0x23, 0x17, 0x9b, 0x2f, 0x58, 0x4e, 0x48, 0x95, 0x8b, 0xaf, 0x28,
	0xb5, 0xb0, 0x34, 0xb3, 0x28, 0x35, 0x25, 0xbe, 0x2c, 0x31, 0xa7, 0x34, 0x55, 0x82, 0x51, 0x81,
	0x49, 0x83, 0x35, 0x88, 0x17, 0x26, 0x1a, 0x06, 0x12, 0x14, 0xd2, 0xe5, 0x62, 0x4d, 0xce, 0xcf,
	0xc9, 0x2f, 0x92, 0x60, 0x52, 0x60, 0xd4, 0xe0, 0x33, 0x12, 0xd7, 0x83, 0x1b, 0xa6, 0x07, 0x31,
	0x48, 0xcf, 0x19, 0x24, 0x1d, 0x04, 0x51, 0xa5, 0xa4, 0xca, 0xc5, 0x0a, 0xe6, 0x0b, 0x71, 0x70,
	0xb1, 0x38, 0xf9, 0x84, 0xba, 0x0a, 0x30, 0x0a, 0x71, 0x72, 0xb1, 0xba, 0x07, 0xb9, 0xba, 0xfa,
	0x09, 0x30, 0x09, 0xb1, 0x73, 0x31, 0x07, 0xb9, 0xba, 0x08, 0x30, 0x3b, 0x39, 0x47, 0x39, 0xa6,
	0x67, 0x96, 0x64, 0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0xa7, 0xe7, 0xe7, 0x24, 0xe6, 0xa5,
	0xeb, 0x83, 0x5d, 0x9b, 0x54, 0x9a, 0x06, 0x61, 0x24, 0xeb, 0xa6, 0xa7, 0xe6, 0xe9, 0xa6, 0xe7,
	0xeb, 0x83, 0xec, 0x4a, 0x49, 0x2c, 0x49, 0x84, 0x78, 0xca, 0x1a, 0x6e, 0x3f, 0x20, 0x00, 0x00,
	0xff, 0xff, 0x49, 0x3b, 0x52, 0x44, 0xec, 0x00, 0x00, 0x00,
}
