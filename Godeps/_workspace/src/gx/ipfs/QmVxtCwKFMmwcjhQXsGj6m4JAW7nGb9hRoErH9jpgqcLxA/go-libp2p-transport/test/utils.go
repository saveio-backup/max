package utils

import (
	"fmt"
	"io"
	"testing"

	tpt "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVxtCwKFMmwcjhQXsGj6m4JAW7nGb9hRoErH9jpgqcLxA/go-libp2p-transport"
	ma "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmWWQ2Txc2c6tqjsBpzg5Ar652cHPGNsQQp2SejkNmkUMb/go-multiaddr"
)

func SubtestTransport(t *testing.T, ta, tb tpt.Transport, addr string) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		t.Fatal(err)
	}

	list, err := ta.Listen(maddr)
	if err != nil {
		t.Fatal(err)
	}

	dialer, err := tb.Dialer(list.Multiaddr())
	if err != nil {
		t.Fatal(err)
	}

	accepted := make(chan tpt.Conn, 1)
	errs := make(chan error, 1)
	go func() {
		b, err := list.Accept()
		if err != nil {
			errs <- err
			return
		}

		accepted <- b
	}()

	a, err := dialer.Dial(list.Multiaddr())
	if err != nil {
		t.Fatal(err)
	}

	var b tpt.Conn
	select {
	case b = <-accepted:
	case err := <-errs:
		t.Fatal(err)
	}

	defer a.Close()
	defer b.Close()

	err = checkDataTransfer(a, b)
	if err != nil {
		t.Fatal(err)
	}

}

func checkDataTransfer(a, b io.ReadWriter) error {
	errs := make(chan error, 2)
	data := []byte("this is some test data")

	go func() {
		n, err := a.Write(data)
		if err != nil {
			errs <- err
			return
		}

		if n != len(data) {
			errs <- fmt.Errorf("failed to write enough data (a->b)")
			return
		}

		buf := make([]byte, len(data))
		_, err = io.ReadFull(a, buf)
		if err != nil {
			errs <- err
			return
		}

		errs <- nil
	}()

	go func() {
		buf := make([]byte, len(data))
		_, err := io.ReadFull(b, buf)
		if err != nil {
			errs <- err
			return
		}

		n, err := b.Write(data)
		if err != nil {
			errs <- err
			return
		}

		if n != len(data) {
			errs <- fmt.Errorf("failed to write enough data (b->a)")
			return
		}

		errs <- nil
	}()

	err := <-errs
	if err != nil {
		return err
	}
	err = <-errs
	if err != nil {
		return err
	}

	return nil
}
