package v1

import (
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"
)

type mergoOptArr []func(*mergo.Config)

func (arr mergoOptArr) names() []string {
	names := make([]string, len(arr))
	for i, opt := range arr {
		names[i] = runtime.FuncForPC(reflect.ValueOf(opt).Pointer()).Name()
	}
	sort.Strings(names)
	return names
}

func TestMergoConfiguration(t *testing.T) {
	valTrue := true
	tests := map[string]struct {
		mo   *MergeOptions
		want mergoOptArr
	}{
		"DefaultOptionsNil": {
			want: mergoOptArr{
				mergo.WithOverride,
			},
		},
		"DefaultOptionsEmptyStruct": {
			mo: &MergeOptions{},
			want: mergoOptArr{
				mergo.WithOverride,
			},
		},
		"MapKeepOnly": {
			mo: &MergeOptions{
				KeepMapValues: &valTrue,
			},
			want: mergoOptArr{},
		},
		"AppendSliceOnly": {
			mo: &MergeOptions{
				AppendSlice: &valTrue,
			},
			want: mergoOptArr{
				mergo.WithAppendSlice,
				mergo.WithOverride,
			},
		},
		"MapKeepAppendSlice": {
			mo: &MergeOptions{
				AppendSlice:   &valTrue,
				KeepMapValues: &valTrue,
			},
			want: mergoOptArr{
				mergo.WithAppendSlice,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want.names(), mergoOptArr(tc.mo.MergoConfiguration()).names()); diff != "" {
				t.Errorf("\nmo.MergoConfiguration(): -want, +got:\n %s", diff)
			}

		})
	}
}
