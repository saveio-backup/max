// Code generated by protoc-gen-go.
// source: test.proto
// DO NOT EDIT!

/*
Package test is a generated protocol buffer package.

It is generated from these files:
	test.proto

It has these top-level messages:
	Foo
	Bar
*/
package test

import proto "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Foo struct {
	A *int32  `protobuf:"varint,1,opt,name=a" json:"a,omitempty"`
	B *int32  `protobuf:"varint,2,opt,name=b" json:"b,omitempty"`
	C []int32 `protobuf:"varint,3,rep,name=c" json:"c,omitempty"`
	D *int32  `protobuf:"varint,4,opt,name=d" json:"d,omitempty"`
}

func (m *Foo) Reset()         { *m = Foo{} }
func (m *Foo) String() string { return proto.CompactTextString(m) }
func (*Foo) ProtoMessage()    {}

func (m *Foo) GetA() int32 {
	if m != nil && m.A != nil {
		return *m.A
	}
	return 0
}

func (m *Foo) GetB() int32 {
	if m != nil && m.B != nil {
		return *m.B
	}
	return 0
}

func (m *Foo) GetC() []int32 {
	if m != nil {
		return m.C
	}
	return nil
}

func (m *Foo) GetD() int32 {
	if m != nil && m.D != nil {
		return *m.D
	}
	return 0
}

type Bar struct {
	Foos []*Foo   `protobuf:"bytes,1,rep,name=foos" json:"foos,omitempty"`
	Strs []string `protobuf:"bytes,2,rep,name=strs" json:"strs,omitempty"`
	Bufs [][]byte `protobuf:"bytes,3,rep,name=bufs" json:"bufs,omitempty"`
}

func (m *Bar) Reset()         { *m = Bar{} }
func (m *Bar) String() string { return proto.CompactTextString(m) }
func (*Bar) ProtoMessage()    {}

func (m *Bar) GetFoos() []*Foo {
	if m != nil {
		return m.Foos
	}
	return nil
}

func (m *Bar) GetStrs() []string {
	if m != nil {
		return m.Strs
	}
	return nil
}

func (m *Bar) GetBufs() [][]byte {
	if m != nil {
		return m.Bufs
	}
	return nil
}

func init() {
}
