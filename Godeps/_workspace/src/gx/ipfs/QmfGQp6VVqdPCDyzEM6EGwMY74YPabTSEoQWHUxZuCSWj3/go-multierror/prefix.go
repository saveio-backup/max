package multierror

import (
	"fmt"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/Qmbg4PQLEvf2XW8vrai9STFDerV7kttkfKcVdkoRf9Z7Xu/go-errwrap"
)

// Prefix is a helper function that will prefix some text
// to the given error. If the error is a multierror.Error, then
// it will be prefixed to each wrapped error.
//
// This is useful to use when appending multiple multierrors
// together in order to give better scoping.
func Prefix(err error, prefix string) error {
	if err == nil {
		return nil
	}

	format := fmt.Sprintf("%s {{err}}", prefix)
	switch err := err.(type) {
	case *Error:
		// Typed nils can reach here, so initialize if we are nil
		if err == nil {
			err = new(Error)
		}

		// Wrap each of the errors
		for i, e := range err.Errors {
			err.Errors[i] = errwrap.Wrapf(format, e)
		}

		return err
	default:
		return errwrap.Wrapf(format, err)
	}
}
