package iface

import "errors"

var (
	ErrIsDir   = errors.New("object is a directory")
	ErrOffline = errors.New("can't resolve, ont-ipfs node is offline")
)
