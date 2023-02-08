package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krateoplatformops/provider-runtime/pkg/test"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAPIPatchingApplicator(t *testing.T) {
	errBoom := errors.New("boom")
	desired := &object{}
	desired.SetName("desired")

	type args struct {
		ctx context.Context
		o   client.Object
		ao  []ApplyOption
	}

	type want struct {
		o   client.Object
		err error
	}

	cases := map[string]struct {
		reason string
		c      client.Client
		args   args
		want   want
	}{
		"GetError": {
			reason: "An error should be returned if we can't get the object",
			c:      &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot get object"),
			},
		},
		"CreateError": {
			reason: "No error should be returned if we successfully create a new object",
			c: &test.MockClient{
				MockGet:    test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, "")),
				MockCreate: test.NewMockCreateFn(errBoom),
			},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot create object"),
			},
		},
		"ApplyOptionError": {
			reason: "Any errors from an apply option should be returned",
			c:      &test.MockClient{MockGet: test.NewMockGetFn(nil)},
			args: args{
				o:  &object{},
				ao: []ApplyOption{func(_ context.Context, _, _ runtime.Object) error { return errBoom }},
			},
			want: want{
				o:   &object{},
				err: errBoom,
			},
		},
		"PatchError": {
			reason: "An error should be returned if we can't patch the object",
			c: &test.MockClient{
				MockGet:   test.NewMockGetFn(nil),
				MockPatch: test.NewMockPatchFn(errBoom),
			},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot patch object"),
			},
		},
		"Created": {
			reason: "No error should be returned if we successfully create a new object",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, "")),
				MockCreate: test.NewMockCreateFn(nil, func(o client.Object) error {
					*o.(*object) = *desired
					return nil
				}),
			},
			args: args{
				o: desired,
			},
			want: want{
				o: desired,
			},
		},
		"Patched": {
			reason: "No error should be returned if we successfully patch an existing object",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(nil),
				MockPatch: test.NewMockPatchFn(nil, func(o client.Object) error {
					*o.(*object) = *desired
					return nil
				}),
			},
			args: args{
				o: desired,
			},
			want: want{
				o: desired,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			a := NewAPIPatchingApplicator(tc.c)
			err := a.Apply(tc.args.ctx, tc.args.o, tc.args.ao...)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nApply(...): -want error, +got error\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, tc.args.o); diff != "" {
				t.Errorf("\n%s\nApply(...): -want, +got\n%s\n", tc.reason, diff)
			}
		})
	}
}

type object struct {
	runtime.Object
	metav1.ObjectMeta
}

func (o *object) DeepCopyObject() runtime.Object {
	return &object{ObjectMeta: *o.ObjectMeta.DeepCopy()}
}
