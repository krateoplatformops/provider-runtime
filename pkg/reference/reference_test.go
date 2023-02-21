package reference

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rpv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	"github.com/krateoplatformops/provider-runtime/pkg/resource/fake"
	"github.com/krateoplatformops/provider-runtime/pkg/test"
)

// its contemporaries in pkg/resource/fake because it would cause an import
// cycle.
type FakeManagedList struct {
	client.ObjectList

	Items []resource.Managed
}

func (fml *FakeManagedList) GetItems() []resource.Managed {
	return fml.Items
}

func TestToAndFromPtr(t *testing.T) {
	cases := map[string]struct {
		want string
	}{
		"Zero":    {want: ""},
		"NonZero": {want: "pointy"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := FromPtrValue(ToPtrValue(tc.want))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FromPtrValue(ToPtrValue(%s): -want, +got: %s", tc.want, diff)

			}
		})

	}
}

func TestToAndFromPtrValues(t *testing.T) {
	cases := map[string]struct {
		want []string
	}{
		"Nil":      {want: []string{}},
		"Zero":     {want: []string{""}},
		"NonZero":  {want: []string{"pointy"}},
		"Multiple": {want: []string{"pointy", "pointers"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := FromPtrValues(ToPtrValues(tc.want))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FromPtrValues(ToPtrValues(%s): -want, +got: %s", tc.want, diff)

			}
		})
	}
}

func TestResolve(t *testing.T) {
	errBoom := errors.New("boom")
	now := metav1.Now()
	value := "coolv"
	ref := &rpv1.Reference{Name: "cool"}
	optionalPolicy := rpv1.ResolutionPolicyOptional
	alwaysPolicy := rpv1.ResolvePolicyAlways
	optionalRef := &rpv1.Reference{Name: "cool", Policy: &rpv1.Policy{Resolution: &optionalPolicy}}
	alwaysRef := &rpv1.Reference{Name: "cool", Policy: &rpv1.Policy{Resolve: &alwaysPolicy}}

	controlled := &fake.Managed{}
	controlled.SetName(value)
	meta.SetExternalName(controlled, value)
	meta.AddControllerReference(controlled, meta.AsController(&rpv1.TypedReference{UID: types.UID("very-unique")}))

	type args struct {
		ctx context.Context
		req ResolutionRequest
	}
	type want struct {
		rsp ResolutionResponse
		err error
	}
	cases := map[string]struct {
		reason string
		c      client.Reader
		from   resource.Managed
		args   args
		want   want
	}{
		"FromDeleted": {
			reason: "Should return early if the referencing managed resource was deleted",
			from:   &fake.Managed{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}},
			args: args{
				req: ResolutionRequest{},
			},
			want: want{
				rsp: ResolutionResponse{},
				err: nil,
			},
		},
		"AlreadyResolved": {
			reason: "Should return early if the current value is non-zero",
			from:   &fake.Managed{},
			args: args{
				req: ResolutionRequest{CurrentValue: value},
			},
			want: want{
				rsp: ResolutionResponse{ResolvedValue: value},
				err: nil,
			},
		},
		"AlwaysResolveReference": {
			reason: "Should not return early if the current value is non-zero, when the resolve policy is set to" +
				"Always",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					meta.SetExternalName(obj.(metav1.Object), value)
					return nil
				}),
			},
			from: &fake.Managed{},
			args: args{
				req: ResolutionRequest{
					Reference:    alwaysRef,
					To:           &fake.Managed{},
					Extract:      ExternalName(),
					CurrentValue: "oldValue",
				},
			},
			want: want{
				rsp: ResolutionResponse{
					ResolvedValue:     value,
					ResolvedReference: alwaysRef,
				},
				err: nil,
			},
		},
		"Unresolvable": {
			reason: "Should return early if neither a reference or selector were provided",
			from:   &fake.Managed{},
			args: args{
				req: ResolutionRequest{},
			},
			want: want{
				err: nil,
			},
		},
		"GetError": {
			reason: "Should return errors encountered while getting the referenced resource",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(errBoom),
			},
			from: &fake.Managed{},
			args: args{
				req: ResolutionRequest{
					Reference: ref,
					To:        &fake.Managed{},
					Extract:   ExternalName(),
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetManaged),
			},
		},
		"ResolvedNoValue": {
			reason: "Should return an error if the extract function returns the empty string",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(nil),
			},
			from: &fake.Managed{},
			args: args{
				req: ResolutionRequest{
					Reference: ref,
					To:        &fake.Managed{},
					Extract:   func(resource.Object) string { return "" },
				},
			},
			want: want{
				rsp: ResolutionResponse{
					ResolvedReference: ref,
				},
				err: errors.New(errNoValue),
			},
		},
		"SuccessfulResolve": {
			reason: "No error should be returned when the value is successfully extracted",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
					meta.SetExternalName(obj.(metav1.Object), value)
					return nil
				}),
			},
			from: &fake.Managed{},
			args: args{
				req: ResolutionRequest{
					Reference: ref,
					To:        &fake.Managed{},
					Extract:   ExternalName(),
				},
			},
			want: want{
				rsp: ResolutionResponse{
					ResolvedValue:     value,
					ResolvedReference: ref,
				},
			},
		},
		"OptionalReference": {
			reason: "No error should be returned when the resolution policy is Optional",
			c: &test.MockClient{
				MockGet: test.NewMockGetFn(nil),
			},
			from: &fake.Managed{},
			args: args{
				req: ResolutionRequest{
					Reference: optionalRef,
					To:        &fake.Managed{},
					Extract:   func(resource.Object) string { return "" },
				},
			},
			want: want{
				rsp: ResolutionResponse{
					ResolvedReference: optionalRef,
				},
				err: nil,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewAPIResolver(tc.c, tc.from)
			got, err := r.Resolve(tc.args.ctx, tc.args.req)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nControllersMustMatch(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.rsp, got); diff != "" {
				t.Errorf("\n%s\nControllersMustMatch(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
