package mc_msgpack

import (
	"io"

	gocodec "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVTAmbCaPqdfbmpWDCJMQNFxbyJoG2USFsumXmTWY5LFp/go-codec/codec"
	mc "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmewJ1Zp9Hwz5HcMd7JYjhLXwvEHTL2UBCCz3oLt1E2N5z/go-multicodec"
)

const HeaderPath = "/msgpack"

var Header = mc.Header([]byte(HeaderPath))

type codec struct {
	mc     bool
	handle *gocodec.MsgpackHandle
}

func Codec(h *gocodec.MsgpackHandle) mc.Codec {
	return &codec{
		mc:     false,
		handle: h,
	}
}

func DefaultMsgpackHandle() *gocodec.MsgpackHandle {
	return &gocodec.MsgpackHandle{}
}

func Multicodec(h *gocodec.MsgpackHandle) mc.Multicodec {
	return &codec{
		mc:     true,
		handle: h,
	}
}

func (c *codec) Encoder(w io.Writer) mc.Encoder {
	return &encoder{
		w:   w,
		mc:  c.mc,
		enc: gocodec.NewEncoder(w, c.handle),
	}
}

func (c *codec) Decoder(r io.Reader) mc.Decoder {
	return &decoder{
		r:   r,
		mc:  c.mc,
		dec: gocodec.NewDecoder(r, c.handle),
	}
}

func (c *codec) Header() []byte {
	return Header
}

type encoder struct {
	w   io.Writer
	mc  bool
	enc *gocodec.Encoder
}

type decoder struct {
	r   io.Reader
	mc  bool
	dec *gocodec.Decoder
}

func (c *encoder) Encode(v interface{}) error {
	// if multicodec, write the header first
	if c.mc {
		if _, err := c.w.Write(Header); err != nil {
			return err
		}
	}
	return c.enc.Encode(v)
}

func (c *decoder) Decode(v interface{}) error {
	// if multicodec, consume the header first
	if c.mc {
		if err := mc.ConsumeHeader(c.r, Header); err != nil {
			return err
		}
	}
	return c.dec.Decode(v)
}
