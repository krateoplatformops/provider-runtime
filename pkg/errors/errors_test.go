package errors

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWrap(t *testing.T) {
	type args struct {
		err     error
		message string
	}
	cases := map[string]struct {
		args args
		want error
	}{
		"NilError": {
			args: args{
				err:     nil,
				message: "very useful context",
			},
			want: nil,
		},
		"NonNilError": {
			args: args{
				err:     New("boom"),
				message: "very useful context",
			},
			want: Errorf("very useful context: %w", New("boom")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := Wrap(tc.args.err, tc.args.message)
			if diff := cmp.Diff(tc.want, got, EquateErrors()); diff != "" {
				t.Errorf("Wrap(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	type args struct {
		err     error
		message string
		args    []any
	}
	cases := map[string]struct {
		args args
		want error
	}{
		"NilError": {
			args: args{
				err:     nil,
				message: "very useful context",
			},
			want: nil,
		},
		"NonNilError": {
			args: args{
				err:     New("boom"),
				message: "very useful context about %s",
				args:    []any{"ducks"},
			},
			want: Errorf("very useful context about %s: %w", "ducks", New("boom")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := Wrapf(tc.args.err, tc.args.message, tc.args.args...)
			if diff := cmp.Diff(tc.want, got, EquateErrors()); diff != "" {
				t.Errorf("Wrapf(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestCause(t *testing.T) {
	cases := map[string]struct {
		err  error
		want error
	}{
		"NilError": {
			err:  nil,
			want: nil,
		},
		"BareError": {
			err:  New("boom"),
			want: New("boom"),
		},
		"WrappedError": {
			err:  Wrap(Wrap(New("boom"), "interstitial context"), "very important context"),
			want: New("boom"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := Cause(tc.err)
			if diff := cmp.Diff(tc.want, got, EquateErrors()); diff != "" {
				t.Errorf("Cause(...): -want, +got:\n%s", diff)
			}
		})
	}
}

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
