// +build !darwin

package syscallx

// This file just contains wrappers for platforms that already have
// the right stuff in golang.org/x/sys/unix.

import (
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmVGjyM9i2msKvLXwh9VosCTgP4mL91kC7hDmqnwTTx6Hu/sys/unix"
)

func Getxattr(path string, attr string, dest []byte) (sz int, err error) {
	return unix.Getxattr(path, attr, dest)
}

func Listxattr(path string, dest []byte) (sz int, err error) {
	return unix.Listxattr(path, dest)
}

func Setxattr(path string, attr string, data []byte, flags int) (err error) {
	return unix.Setxattr(path, attr, data, flags)
}

func Removexattr(path string, attr string) (err error) {
	return unix.Removexattr(path, attr)
}
