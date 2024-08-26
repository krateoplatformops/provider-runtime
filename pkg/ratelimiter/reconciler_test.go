package ratelimiter

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ ratelimiter.RateLimiter = &predictableRateLimiter{}

type predictableRateLimiter struct{ d time.Duration }

func (r *predictableRateLimiter) When(_ any) time.Duration { return r.d }
func (r *predictableRateLimiter) Forget(_ any)             {}
func (r *predictableRateLimiter) NumRequeues(_ any) int    { return 0 }

func TestReconcile(t *testing.T) {
	type args struct {
		ctx context.Context
		req reconcile.Request
	}
	type want struct {
		res reconcile.Result
		err error
	}

	cases := map[string]struct {
		reason string
		r      reconcile.Reconciler
		args   args
		want   want
	}{
		"NotRateLimited": {
			reason: "Requests that are not rate limited should be passed to the inner Reconciler.",
			r: New("test",
				reconcile.Func(func(c context.Context, r reconcile.Request) (reconcile.Result, error) {
					return reconcile.Result{Requeue: true}, nil
				}),
				&predictableRateLimiter{}),
			want: want{
				res: reconcile.Result{Requeue: true},
				err: nil,
			},
		},
		"RateLimited": {
			reason: "Requests that are rate limited should be requeued after the duration specified by the RateLimiter.",
			r:      New("test", nil, &predictableRateLimiter{d: 8 * time.Second}),
			want: want{
				res: reconcile.Result{RequeueAfter: 8 * time.Second},
				err: nil,
			},
		},
		"Returning": {
			reason: "Returning requests that were previously rate limited should be allowed through without further rate limiting.",
			r: func() reconcile.Reconciler {
				inner := reconcile.Func(func(c context.Context, r reconcile.Request) (reconcile.Result, error) {
					return reconcile.Result{Requeue: true}, nil
				})

				// Rate limit the request once.
				r := New("test", inner, &predictableRateLimiter{d: 8 * time.Second})
				r.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "limited"}})
				return r

			}(),
			args: args{
				ctx: context.Background(),
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "limited"}},
			},
			want: want{
				res: reconcile.Result{Requeue: true},
				err: nil,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := tc.r.Reconcile(tc.args.ctx, tc.args.req)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("%s\nr.Reconcile(...): -want, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.res, got); diff != "" {
				t.Errorf("%s\nr.Reconcile(...): -want, +got result:\n%s", tc.reason, diff)
			}
		})
	}
}

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
