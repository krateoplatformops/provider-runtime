// Package event records Kubernetes events.
package event

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSliceMap(t *testing.T) {

	type args struct {
		from []string
		to   map[string]string
	}
	cases := map[string]struct {
		reason string
		args   args
		want   map[string]string
	}{
		"OnePair": {
			reason: "One key value pair should be added.",
			args: args{
				from: []string{"key", "val"},
				to:   map[string]string{},
			},
			want: map[string]string{"key": "val"},
		},
		"TwoPairs": {
			reason: "Two key value pairs should be added.",
			args: args{
				from: []string{
					"key", "val",
					"another", "value",
				},
				to: map[string]string{},
			},
			want: map[string]string{
				"key":     "val",
				"another": "value",
			},
		},
		"NoValue": {
			reason: "Two key value pairs should be added.",
			args: args{
				from: []string{"key"},
				to:   map[string]string{},
			},
			want: map[string]string{},
		},
		"ExtraneousKey": {
			reason: "One key value pair should be added.",
			args: args{
				from: []string{
					"key", "val",
					"extraneous",
				},
				to: map[string]string{},
			},
			want: map[string]string{"key": "val"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sliceMap(tc.args.from, tc.args.to)

			if diff := cmp.Diff(tc.want, tc.args.to); diff != "" {
				t.Errorf("%s\nsliceMap(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}

}
