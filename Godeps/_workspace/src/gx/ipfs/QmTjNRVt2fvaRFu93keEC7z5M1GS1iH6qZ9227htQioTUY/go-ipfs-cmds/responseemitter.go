package cmds

import (
	"fmt"
	"io"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmceUdzxkimdYsgtX733uNgzf1DLHyBKN6ehGSp85ayppM/go-ipfs-cmdkit"
)

// Single can be used to signal to any ResponseEmitter that only one value will be emitted.
// This is important e.g. for the http.ResponseEmitter so it can set the HTTP headers appropriately.
type Single struct {
	Value interface{}
}

func (s Single) String() string {
	return fmt.Sprintf("Single{%v}", s.Value)
}

func (s Single) GoString() string {
	return fmt.Sprintf("Single{%#v}", s.Value)
}

// EmitOnce is a helper that emits a value wrapped in Single, to signal that this will be the only value sent.
func EmitOnce(re ResponseEmitter, v interface{}) error {
	return re.Emit(Single{v})
}

// ResponseEmitter encodes and sends the command code's output to the client.
// It is all a command can write to.
type ResponseEmitter interface {
	// closes http conn or channel
	io.Closer

	// SetLength sets the length of the output
	// err is an interface{} so we don't have to manually convert to error.
	SetLength(length uint64)

	// SetError sets the response error
	// err is an interface{} so we don't have to manually convert to error.
	SetError(err interface{}, code cmdkit.ErrorType)

	// Emit sends a value
	// if value is io.Reader we just copy that to the connection
	// other values are marshalled
	Emit(value interface{}) error
}

type EncodingEmitter interface {
	ResponseEmitter

	SetEncoder(func(io.Writer) Encoder)
}

type Header interface {
	Head() Head
}

// Copy sends all values received on res to re. If res is closed, it closes re.
func Copy(re ResponseEmitter, res Response) error {
	re.SetLength(res.Length())

	for {
		v, err := res.RawNext()
		switch err {
		case nil:
			// all good, go on
		case io.EOF:
			re.Close()
			return nil
		default:
			return err
		}

		err = re.Emit(v)
		if err != nil {
			return err
		}
	}
}
