// Protocol Buffers for Go with Gadgets
//
// Copyright (c) 2013, The GoGo Authors. All rights reserved.
// http://github.com/gogo/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package equal

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/gogoproto"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/plugin/testgen"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/protoc-gen-gogo/generator"
)

type test struct {
	*generator.Generator
}

func NewTest(g *generator.Generator) testgen.TestPlugin {
	return &test{g}
}

func (p *test) Generate(imports generator.PluginImports, file *generator.FileDescriptor) bool {
	used := false
	randPkg := imports.NewImport("math/rand")
	timePkg := imports.NewImport("time")
	testingPkg := imports.NewImport("testing")
	protoPkg := imports.NewImport("github.com/gogo/protobuf/proto")
	unsafePkg := imports.NewImport("unsafe")
	if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
		protoPkg = imports.NewImport("github.com/golang/protobuf/proto")
	}
	for _, message := range file.Messages() {
		ccTypeName := generator.CamelCaseSlice(message.TypeName())
		if !gogoproto.HasVerboseEqual(file.FileDescriptorProto, message.DescriptorProto) {
			continue
		}
		if message.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}

		if gogoproto.HasTestGen(file.FileDescriptorProto, message.DescriptorProto) {
			used = true
			hasUnsafe := gogoproto.IsUnsafeMarshaler(file.FileDescriptorProto, message.DescriptorProto) ||
				gogoproto.IsUnsafeUnmarshaler(file.FileDescriptorProto, message.DescriptorProto)
			p.P(`func Test`, ccTypeName, `VerboseEqual(t *`, testingPkg.Use(), `.T) {`)
			p.In()
			if hasUnsafe {
				if hasUnsafe {
					p.P(`var bigendian uint32 = 0x01020304`)
					p.P(`if *(*byte)(`, unsafePkg.Use(), `.Pointer(&bigendian)) == 1 {`)
					p.In()
					p.P(`t.Skip("unsafe does not work on big endian architectures")`)
					p.Out()
					p.P(`}`)
				}
			}
			p.P(`popr := `, randPkg.Use(), `.New(`, randPkg.Use(), `.NewSource(`, timePkg.Use(), `.Now().UnixNano()))`)
			p.P(`p := NewPopulated`, ccTypeName, `(popr, false)`)
			p.P(`dAtA, err := `, protoPkg.Use(), `.Marshal(p)`)
			p.P(`if err != nil {`)
			p.In()
			p.P(`panic(err)`)
			p.Out()
			p.P(`}`)
			p.P(`msg := &`, ccTypeName, `{}`)
			p.P(`if err := `, protoPkg.Use(), `.Unmarshal(dAtA, msg); err != nil {`)
			p.In()
			p.P(`panic(err)`)
			p.Out()
			p.P(`}`)
			p.P(`if err := p.VerboseEqual(msg); err != nil {`)
			p.In()
			p.P(`t.Fatalf("%#v !VerboseEqual %#v, since %v", msg, p, err)`)
			p.Out()
			p.P(`}`)
			p.Out()
			p.P(`}`)
		}

	}
	return used
}

func init() {
	testgen.RegisterTestPlugin(NewTest)
}
