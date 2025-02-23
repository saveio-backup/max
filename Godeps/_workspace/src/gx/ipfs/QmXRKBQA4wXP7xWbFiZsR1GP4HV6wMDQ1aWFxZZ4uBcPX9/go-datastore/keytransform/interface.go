package keytransform

import ds "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmXRKBQA4wXP7xWbFiZsR1GP4HV6wMDQ1aWFxZZ4uBcPX9/go-datastore"

// KeyMapping is a function that maps one key to annother
type KeyMapping func(ds.Key) ds.Key

// KeyTransform is an object with a pair of functions for (invertibly)
// transforming keys
type KeyTransform interface {
	ConvertKey(ds.Key) ds.Key
	InvertKey(ds.Key) ds.Key
}

// Datastore is a keytransform.Datastore
type Datastore interface {
	ds.Shim
	KeyTransform
}

// Wrap wraps a given datastore with a KeyTransform function.
// The resulting wrapped datastore will use the transform on all Datastore
// operations.
func Wrap(child ds.Datastore, t KeyTransform) *ktds {
	if t == nil {
		panic("t (KeyTransform) is nil")
	}

	if child == nil {
		panic("child (ds.Datastore) is nil")
	}

	return &ktds{child: child, KeyTransform: t}
}
