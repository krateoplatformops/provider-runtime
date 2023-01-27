package test

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// EquateErrors returns true if the supplied errors are of the same type and
// produce identical strings. This mirrors the error comparison behaviour of
// https://github.com/go-test/deep, which most Crossplane tests targeted before
// we switched to go-cmp.
//
// This differs from cmpopts.EquateErrors, which does not test for error strings
// and instead returns whether one error 'is' (in the errors.Is sense) the
// other.
func EquateErrors() cmp.Option {
	return cmp.Comparer(func(a, b error) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}

		av := reflect.ValueOf(a)
		bv := reflect.ValueOf(b)
		if av.Type() != bv.Type() {
			return false
		}

		return a.Error() == b.Error()
	})
}

// EquateConditions sorts any slices of Condition before comparing them.
func EquateConditions() cmp.Option {
	return cmpopts.SortSlices(func(i, j prv1.Condition) bool { return i.Type < j.Type })
}
