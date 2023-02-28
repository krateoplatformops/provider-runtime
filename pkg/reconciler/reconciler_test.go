package reconciler

import (
	"context"
	"testing"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
	"github.com/krateoplatformops/provider-runtime/pkg/resource/fake"
	"github.com/krateoplatformops/provider-runtime/pkg/test"
)

var _ reconcile.Reconciler = &Reconciler{}

func TestReconciler(t *testing.T) {
	type args struct {
		m  manager.Manager
		mg resource.ManagedKind
		o  []ReconcilerOption
	}

	type want struct {
		result reconcile.Result
		err    error
	}

	errBoom := errors.New("boom")
	now := metav1.Now()

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"GetManagedError": {
			reason: "Any error (except not found) encountered while getting the resource under reconciliation should be returned.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(errBoom)},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{err: errors.Wrap(errBoom, errGetManaged)},
		},
		"ManagedNotFound": {
			reason: "Not found errors encountered while getting the resource under reconciliation should be ignored.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{MockGet: test.NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, ""))},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{result: reconcile.Result{}},
		},
		"RemoveFinalizerErrorDeletionPolicyOrphan": {
			reason: "Errors removing the managed resource finalizer should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionOrphan)
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetDeletionTimestamp(&now)
							want.SetDeletionPolicy(prv1.DeletionOrphan)
							want.SetConditions(prv1.Deleting())
							want.SetConditions(prv1.ReconcileError(errBoom))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors removing the managed resource finalizer should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithFinalizer(resource.FinalizerFns{RemoveFinalizerFn: func(_ context.Context, _ resource.Object) error { return errBoom }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"DeleteSuccessfulDeletionPolicyOrphan": {
			reason: "Successful managed resource deletion with deletion policy Orphan should not trigger a requeue or status update.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionOrphan)
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithFinalizer(resource.FinalizerFns{RemoveFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: false}},
		},
		"ExternalCreatePending": {
			reason: "We should return early if the managed resource appears to be pending creation. We might have leaked a resource and don't want to create another.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							meta.SetExternalCreatePending(obj, now.Time)
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							meta.SetExternalCreatePending(want, now.Time)
							want.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.New(errCreateIncomplete)))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "We should update our status when we're asked to reconcile a managed resource that is pending creation."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{result: reconcile.Result{Requeue: false}},
		},
		"ResolveReferencesError": {
			reason: "Errors during reference resolution references should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errBoom))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors during reference resolution should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"ExternalConnectError": {
			reason: "Errors connecting to the provider should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, got client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errReconcileConnect)))
							if diff := cmp.Diff(want, got, test.EquateConditions()); diff != "" {
								reason := "Errors connecting to the provider should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						return nil, errBoom
					})),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"ExternalDisconnectError": {
			reason: "Error disconnecting from the provider should not trigger requeue.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileSuccess())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "A successful no-op reconcile should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnectDisconnecter(ExternalConnectDisconnecterFns{
						ConnectFn: func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
							c := &ExternalClientFns{
								ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
									return ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
								},
							}
							return c, nil
						},
						DisconnectFn: func(_ context.Context) error { return errBoom },
					}),

					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{RequeueAfter: defaultpollInterval}},
		},
		"ExternalObserveError": {
			reason: "Errors observing the external resource should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errReconcileObserve)))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors observing the managed resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{}, errBoom
							},
						}
						return c, nil
					})),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"CreationGracePeriod": {
			reason: "If our resource appears not to exist during the creation grace period we should return early.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							meta.SetExternalCreateSucceeded(obj, time.Now())
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithCreationGracePeriod(1 * time.Minute),
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
						}
						return c, nil
					})),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"ExternalDeleteError": {
			reason: "Errors deleting the external resource should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionDelete)
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetDeletionTimestamp(&now)
							want.SetDeletionPolicy(prv1.DeletionDelete)
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errReconcileDelete)))
							want.SetConditions(prv1.Deleting())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "An error deleting an external resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true}, nil
							},
							DeleteFn: func(_ context.Context, _ resource.Managed) error {
								return errBoom
							},
						}
						return c, nil
					})),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"ExternalDeleteSuccessful": {
			reason: "A deleted managed resource with the 'delete' reclaim policy should delete its external resource then requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionDelete)
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetDeletionTimestamp(&now)
							want.SetDeletionPolicy(prv1.DeletionDelete)
							want.SetConditions(prv1.ReconcileSuccess())
							want.SetConditions(prv1.Deleting())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "A deleted external resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true}, nil
							},
							DeleteFn: func(_ context.Context, _ resource.Managed) error {
								return nil
							},
						}
						return c, nil
					})),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"RemoveFinalizerErrorDeletionPolicyDelete": {
			reason: "Errors removing the managed resource finalizer should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionDelete)
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetDeletionTimestamp(&now)
							want.SetDeletionPolicy(prv1.DeletionDelete)
							want.SetConditions(prv1.Deleting())
							want.SetConditions(prv1.ReconcileError(errBoom))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors removing the managed resource finalizer should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{RemoveFinalizerFn: func(_ context.Context, _ resource.Object) error { return errBoom }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"DeleteSuccessfulDeletionPolicyDelete": {
			reason: "Successful managed resource deletion with deletion policy Delete should not trigger a requeue or status update.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetDeletionTimestamp(&now)
							mg.SetDeletionPolicy(prv1.DeletionDelete)
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{RemoveFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: false}},
		},
		"AddFinalizerError": {
			reason: "Errors adding a finalizer should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errBoom))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors adding a finalizer should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(&NopConnecter{}),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return errBoom }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"UpdateCreatePendingError": {
			reason: "Errors while updating our external-create-pending annotation should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:    test.NewMockGetFn(nil),
						MockUpdate: test.NewMockUpdateFn(errBoom),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							meta.SetExternalCreatePending(want, time.Now())
							want.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.Wrap(errBoom, errUpdateManaged)))
							if diff := cmp.Diff(want, obj, test.EquateConditions(), cmpopts.EquateApproxTime(1*time.Second)); diff != "" {
								reason := "Errors while creating an external resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
							CreateFn: func(_ context.Context, _ resource.Managed) error {
								return errBoom
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"CreateExternalError": {
			reason: "Errors while creating an external resource should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:    test.NewMockGetFn(nil),
						MockUpdate: test.NewMockUpdateFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							meta.SetExternalCreatePending(want, time.Now())
							meta.SetExternalCreateFailed(want, time.Now())
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errReconcileCreate)))
							want.SetConditions(prv1.Creating())
							if diff := cmp.Diff(want, obj, test.EquateConditions(), cmpopts.EquateApproxTime(1*time.Second)); diff != "" {
								reason := "Errors while creating an external resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
							CreateFn: func(_ context.Context, _ resource.Managed) error {
								return errBoom
							},
						}
						return c, nil
					})),
					// We simulate our critical annotation update failing too here.
					// This is mostly just to exercise the code, which just creates a log and an event.
					WithCriticalAnnotationUpdater(CriticalAnnotationUpdateFn(func(ctx context.Context, o client.Object) error { return errBoom })),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"UpdateCriticalAnnotationsError": {
			reason: "Errors updating critical annotations after creation should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:    test.NewMockGetFn(nil),
						MockUpdate: test.NewMockUpdateFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							meta.SetExternalCreatePending(want, time.Now())
							meta.SetExternalCreateSucceeded(want, time.Now())
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errUpdateManagedAnnotations)))
							want.SetConditions(prv1.Creating())
							if diff := cmp.Diff(want, obj, test.EquateConditions(), cmpopts.EquateApproxTime(1*time.Second)); diff != "" {
								reason := "Errors updating critical annotations after creation should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: false}, nil
							},
							CreateFn: func(_ context.Context, _ resource.Managed) error {
								return nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
					WithCriticalAnnotationUpdater(CriticalAnnotationUpdateFn(func(ctx context.Context, o client.Object) error { return errBoom })),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"CreateSuccessful": {
			reason: "Successful managed resource creation should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:    test.NewMockGetFn(nil),
						MockUpdate: test.NewMockUpdateFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							meta.SetExternalCreatePending(want, time.Now())
							meta.SetExternalCreateSucceeded(want, time.Now())
							want.SetConditions(prv1.ReconcileSuccess())
							want.SetConditions(prv1.Creating())
							if diff := cmp.Diff(want, obj, test.EquateConditions(), cmpopts.EquateApproxTime(1*time.Second)); diff != "" {
								reason := "Successful managed resource creation should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(&NopConnecter{}),
					WithCriticalAnnotationUpdater(CriticalAnnotationUpdateFn(func(ctx context.Context, o client.Object) error { return nil })),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"LateInitializeUpdateError": {
			reason: "Errors updating a managed resource to persist late initialized fields should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet:    test.NewMockGetFn(nil),
						MockUpdate: test.NewMockUpdateFn(errBoom),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errUpdateManaged)))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors updating a managed resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ResourceLateInitialized: true}, nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"ExternalResourceUpToDate": {
			reason: "When the external resource exists and is up to date a requeue should be triggered after a long wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileSuccess())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "A successful no-op reconcile should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{RequeueAfter: defaultpollInterval}},
		},
		"UpdateExternalError": {
			reason: "Errors while updating an external resource should trigger a requeue after a short wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileError(errors.Wrap(errBoom, errReconcileUpdate)))
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "Errors while updating an external resource should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, nil
							},
							UpdateFn: func(_ context.Context, _ resource.Managed) error {
								return errBoom
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{Requeue: true}},
		},
		"UpdateSuccessful": {
			reason: "A successful managed resource update should trigger a requeue after a long wait.",
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetConditions(prv1.ReconcileSuccess())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := "A successful managed resource update should be reported as a conditioned status."
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnecter(ExternalConnectorFn(func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
						c := &ExternalClientFns{
							ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
								return ExternalObservation{ResourceExists: true, ResourceUpToDate: false}, nil
							},
							UpdateFn: func(_ context.Context, _ resource.Managed) error {
								return nil
							},
						}
						return c, nil
					})),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{RequeueAfter: defaultpollInterval}},
		},
		"ReconciliationPausedSuccessful": {
			reason: `If a managed resource has the pause annotation with value "true", there should be no further requeue requests.`,
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetAnnotations(map[string]string{meta.AnnotationKeyReconciliationPaused: "true"})
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetAnnotations(map[string]string{meta.AnnotationKeyReconciliationPaused: "true"})
							want.SetConditions(prv1.ReconcilePaused())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := `If managed resource has the pause annotation with value "true", it should acquire "Synced" status condition with the status "False" and the reason "ReconcilePaused".`
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{result: reconcile.Result{}},
		},
		"ReconciliationResumes": {
			reason: `If a managed resource has the pause annotation with some value other than "true" and the Synced=False/ReconcilePaused status condition, reconciliation should resume with requeueing.`,
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetAnnotations(map[string]string{meta.AnnotationKeyReconciliationPaused: "false"})
							mg.SetConditions(prv1.ReconcilePaused())
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							want := &fake.Managed{}
							want.SetAnnotations(map[string]string{meta.AnnotationKeyReconciliationPaused: "false"})
							want.SetConditions(prv1.ReconcileSuccess())
							if diff := cmp.Diff(want, obj, test.EquateConditions()); diff != "" {
								reason := `Managed resource should acquire Synced=False/ReconcileSuccess status condition after a resume.`
								t.Errorf("\nReason: %s\n-want, +got:\n%s", reason, diff)
							}
							return nil
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
				o: []ReconcilerOption{
					WithExternalConnectDisconnecter(ExternalConnectDisconnecterFns{
						ConnectFn: func(_ context.Context, mg resource.Managed) (ExternalClient, error) {
							c := &ExternalClientFns{
								ObserveFn: func(_ context.Context, _ resource.Managed) (ExternalObservation, error) {
									return ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
								},
							}
							return c, nil
						},
						DisconnectFn: func(_ context.Context) error { return errBoom },
					}),
					WithFinalizer(resource.FinalizerFns{AddFinalizerFn: func(_ context.Context, _ resource.Object) error { return nil }}),
				},
			},
			want: want{result: reconcile.Result{RequeueAfter: defaultpollInterval}},
		},
		"ReconciliationPausedError": {
			reason: `If a managed resource has the pause annotation with value "true" and the status update due to reconciliation being paused fails, error should be reported causing an exponentially backed-off requeue.`,
			args: args{
				m: &fake.Manager{
					Client: &test.MockClient{
						MockGet: test.NewMockGetFn(nil, func(obj client.Object) error {
							mg := obj.(*fake.Managed)
							mg.SetAnnotations(map[string]string{meta.AnnotationKeyReconciliationPaused: "true"})
							return nil
						}),
						MockStatusUpdate: test.MockSubResourceUpdateFn(func(_ context.Context, obj client.Object, _ ...client.SubResourceUpdateOption) error {
							return errBoom
						}),
					},
					Scheme: fake.SchemeWith(&fake.Managed{}),
				},
				mg: resource.ManagedKind(fake.GVK(&fake.Managed{})),
			},
			want: want{err: errors.Wrap(errBoom, errUpdateManagedStatus)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := NewReconciler(tc.args.m, tc.args.mg, tc.args.o...)
			got, err := r.Reconcile(context.Background(), reconcile.Request{})

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want error, +got error:\n%s", tc.reason, diff)
			}

			if diff := cmp.Diff(tc.want.result, got); diff != "" {
				t.Errorf("\nReason: %s\nr.Reconcile(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}
