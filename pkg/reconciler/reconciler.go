package reconciler

import (
	"context"
	"math/rand/v2"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	"github.com/krateoplatformops/provider-runtime/pkg/event"
	"github.com/krateoplatformops/provider-runtime/pkg/logging"
	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	// FinalizerName is the string that is used as finalizer on managed resource
	// objects.
	FinalizerName = "finalizer.managedresource.krateo.io"

	reconcileGracePeriod = 30 * time.Second
	reconcileTimeout     = 1 * time.Minute

	defaultpollInterval = 1 * time.Minute
	defaultGracePeriod  = 30 * time.Second
)

// Error strings.
const (
	errGetManaged               = "cannot get managed resource"
	errUpdateManagedAnnotations = "cannot update managed resource annotations"
	errCreateIncomplete         = "cannot determine creation result - remove the " + meta.AnnotationKeyExternalCreatePending + " annotation if it is safe to proceed"
	errReconcileConnect         = "connect failed"
	errReconcileObserve         = "observe failed"
	errReconcileCreate          = "create failed"
	errReconcileUpdate          = "update failed"
	errReconcileDelete          = "delete failed"
	errExternalResourceNotExist = "external resource does not exist"
)

// Event reasons.
const (
	reasonCannotConnect       event.Reason = "CannotConnectToProvider"
	reasonCannotDisconnect    event.Reason = "CannotDisconnectFromProvider"
	reasonCannotInitialize    event.Reason = "CannotInitializeManagedResource"
	reasonCannotResolveRefs   event.Reason = "CannotResolveResourceReferences"
	reasonCannotObserve       event.Reason = "CannotObserveExternalResource"
	reasonCannotCreate        event.Reason = "CannotCreateExternalResource"
	reasonCannotDelete        event.Reason = "CannotDeleteExternalResource"
	reasonCannotPublish       event.Reason = "CannotPublishConnectionDetails"
	reasonCannotUnpublish     event.Reason = "CannotUnpublishConnectionDetails"
	reasonCannotUpdate        event.Reason = "CannotUpdateExternalResource"
	reasonCannotUpdateManaged event.Reason = "CannotUpdateManagedResource"

	reasonDeleted event.Reason = "DeletedExternalResource"
	reasonCreated event.Reason = "CreatedExternalResource"
	reasonUpdated event.Reason = "UpdatedExternalResource"
	reasonPending event.Reason = "PendingExternalResource"

	reasonReconciliationPaused event.Reason = "ReconciliationPaused"
)

// ControllerName returns the recommended name for controllers that use this
// package to reconcile a particular kind of managed resource.
func ControllerName(kind string) string {
	return "managed/" + strings.ToLower(kind)
}

// A CriticalAnnotationUpdater is used when it is critical that annotations must
// be updated before returning from the Reconcile loop.
type CriticalAnnotationUpdater interface {
	UpdateCriticalAnnotations(ctx context.Context, o client.Object) error
}

// A CriticalAnnotationUpdateFn may be used when it is critical that annotations
// must be updated before returning from the Reconcile loop.
type CriticalAnnotationUpdateFn func(ctx context.Context, o client.Object) error

// UpdateCriticalAnnotations of the supplied object.
func (fn CriticalAnnotationUpdateFn) UpdateCriticalAnnotations(ctx context.Context, o client.Object) error {
	return fn(ctx, o)
}

// An ExternalConnecter produces a new ExternalClient given the supplied
// Managed resource.
type ExternalConnecter interface {
	// Connect to the provider specified by the supplied managed resource and
	// produce an ExternalClient.
	Connect(ctx context.Context, mg resource.Managed) (ExternalClient, error)
}

// An ExternalDisconnecter disconnects from a provider.
type ExternalDisconnecter interface {
	// Disconnect from the provider and close the ExternalClient.
	Disconnect(ctx context.Context) error
}

// A NopDisconnecter converts an ExternalConnecter into an
// ExternalConnectDisconnecter with a no-op Disconnect method.
type NopDisconnecter struct {
	c ExternalConnecter
}

// Connect calls the underlying ExternalConnecter's Connect method.
func (c *NopDisconnecter) Connect(ctx context.Context, mg resource.Managed) (ExternalClient, error) {
	return c.c.Connect(ctx, mg)
}

// Disconnect does nothing. It never returns an error.
func (c *NopDisconnecter) Disconnect(_ context.Context) error {
	return nil
}

// NewNopDisconnecter converts an ExternalConnecter into an
// ExternalConnectDisconnecter with a no-op Disconnect method.
func NewNopDisconnecter(c ExternalConnecter) ExternalConnectDisconnecter {
	return &NopDisconnecter{c}
}

// An ExternalConnectDisconnecter produces a new ExternalClient given the supplied
// Managed resource.
type ExternalConnectDisconnecter interface {
	ExternalConnecter
	ExternalDisconnecter
}

// An ExternalConnectorFn is a function that satisfies the ExternalConnecter
// interface.
type ExternalConnectorFn func(ctx context.Context, mg resource.Managed) (ExternalClient, error)

// Connect to the provider specified by the supplied managed resource and
// produce an ExternalClient.
func (ec ExternalConnectorFn) Connect(ctx context.Context, mg resource.Managed) (ExternalClient, error) {
	return ec(ctx, mg)
}

// An ExternalDisconnectorFn is a function that satisfies the ExternalConnecter
// interface.
type ExternalDisconnectorFn func(ctx context.Context) error

// Disconnect from provider and close the ExternalClient.
func (ed ExternalDisconnectorFn) Disconnect(ctx context.Context) error {
	return ed(ctx)
}

// ExternalConnectDisconnecterFns are functions that satisfy the
// ExternalConnectDisconnecter interface.
type ExternalConnectDisconnecterFns struct {
	ConnectFn    func(ctx context.Context, mg resource.Managed) (ExternalClient, error)
	DisconnectFn func(ctx context.Context) error
}

// Connect to the provider specified by the supplied managed resource and
// produce an ExternalClient.
func (fns ExternalConnectDisconnecterFns) Connect(ctx context.Context, mg resource.Managed) (ExternalClient, error) {
	return fns.ConnectFn(ctx, mg)
}

// Disconnect from the provider and close the ExternalClient.
func (fns ExternalConnectDisconnecterFns) Disconnect(ctx context.Context) error {
	return fns.DisconnectFn(ctx)
}

// An ExternalClient manages the lifecycle of an external resource.
// None of the calls here should be blocking. All of the calls should be
// idempotent. For example, Create call should not return AlreadyExists error
// if it's called again with the same parameters or Delete call should not
// return error if there is an ongoing deletion or resource does not exist.
type ExternalClient interface {
	// Observe the external resource the supplied Managed resource
	// represents, if any. Observe implementations must not modify the
	// external resource, but may update the supplied Managed resource to
	// reflect the state of the external resource. Status modifications are
	// automatically persisted unless ResourceLateInitialized is true - see
	// ResourceLateInitialized for more detail.
	Observe(ctx context.Context, mg resource.Managed) (ExternalObservation, error)

	// Create an external resource per the specifications of the supplied
	// Managed resource. Called when Observe reports that the associated
	// external resource does not exist. Create implementations may update
	// managed resource annotations, and those updates will be persisted.
	// All other updates will be discarded.
	Create(ctx context.Context, mg resource.Managed) error

	// Update the external resource represented by the supplied Managed
	// resource, if necessary. Called unless Observe reports that the
	// associated external resource is up to date.
	Update(ctx context.Context, mg resource.Managed) error

	// Delete the external resource upon deletion of its associated Managed
	// resource. Called when the managed resource has been deleted.
	Delete(ctx context.Context, mg resource.Managed) error
}

// ExternalClientFns are a series of functions that satisfy the ExternalClient
// interface.
type ExternalClientFns struct {
	ObserveFn func(ctx context.Context, mg resource.Managed) (ExternalObservation, error)
	CreateFn  func(ctx context.Context, mg resource.Managed) error
	UpdateFn  func(ctx context.Context, mg resource.Managed) error
	DeleteFn  func(ctx context.Context, mg resource.Managed) error
}

// Observe the external resource the supplied Managed resource represents, if
// any.
func (e ExternalClientFns) Observe(ctx context.Context, mg resource.Managed) (ExternalObservation, error) {
	return e.ObserveFn(ctx, mg)
}

// Create an external resource per the specifications of the supplied Managed
// resource.
func (e ExternalClientFns) Create(ctx context.Context, mg resource.Managed) error {
	return e.CreateFn(ctx, mg)
}

// Update the external resource represented by the supplied Managed resource, if
// necessary.
func (e ExternalClientFns) Update(ctx context.Context, mg resource.Managed) error {
	return e.UpdateFn(ctx, mg)
}

// Delete the external resource upon deletion of its associated Managed
// resource.
func (e ExternalClientFns) Delete(ctx context.Context, mg resource.Managed) error {
	return e.DeleteFn(ctx, mg)
}

// A NopConnecter does nothing.
type NopConnecter struct{}

// Connect returns a NopClient. It never returns an error.
func (c *NopConnecter) Connect(_ context.Context, _ resource.Managed) (ExternalClient, error) {
	return &NopClient{}, nil
}

// A NopClient does nothing.
type NopClient struct{}

// Observe does nothing. It returns an empty ExternalObservation and no error.
func (c *NopClient) Observe(ctx context.Context, mg resource.Managed) (ExternalObservation, error) {
	return ExternalObservation{}, nil
}

// Create does nothing. It returns an empty ExternalCreation and no error.
func (c *NopClient) Create(ctx context.Context, mg resource.Managed) error {
	return nil
}

// Update does nothing. It returns an empty ExternalUpdate and no error.
func (c *NopClient) Update(ctx context.Context, mg resource.Managed) error {
	return nil
}

// Delete does nothing. It never returns an error.
func (c *NopClient) Delete(ctx context.Context, mg resource.Managed) error { return nil }

// An ExternalObservation is the result of an observation of an external
// resource.
type ExternalObservation struct {
	// ResourceExists must be true if a corresponding external resource exists
	// for the managed resource. Typically this is proven by the presence of an
	// external resource of the expected kind whose unique identifier matches
	// the managed resource's external name. The Provider Runtime uses this information to
	// determine whether it needs to create or delete the external resource.
	ResourceExists bool

	// ResourceUpToDate should be true if the corresponding external resource
	// appears to be up-to-date - i.e. updating the external resource to match
	// the desired state of the managed resource would be a no-op. Keep in mind
	// that often only a subset of external resource fields can be updated.
	// The Provider Runtime uses this information to determine whether it needs to update
	// the external resource.
	ResourceUpToDate bool

	// ResourceLateInitialized should be true if the managed resource's spec was
	// updated during its observation. A provider may update a
	// managed resource's spec fields after it is created or updated, as long as
	// the updates are limited to setting previously unset fields, and adding
	// keys to maps. Provider Runtime uses this information to determine whether
	// changes to the spec were made during observation that must be persisted.
	// Note that changes to the spec will be persisted before changes to the
	// status, and that pending changes to the status may be lost when the spec
	// is persisted. Status changes will be persisted by the first subsequent
	// observation that _does not_ late initialize the managed resource, so it
	// is important that Observe implementations do not late initialize the
	// resource every time they are called.
	ResourceLateInitialized bool

	// Diff is a Debug level message that is sent to the reconciler when
	// there is a change in the observed Managed Resource. It is useful for
	// finding where the observed diverges from the desired state.
	// The string should be a cmp.Diff that details the difference.
	Diff string
}

// A Reconciler reconciles managed resources by creating and managing the
// lifecycle of an external resource, i.e. a resource in an external system such
// as a cloud provider API. Each controller must watch the managed resource kind
// for which it is responsible.
type Reconciler struct {
	client     client.Client
	newManaged func() resource.Managed

	pollInterval        time.Duration
	pollIntervalHook    PollIntervalHook
	timeout             time.Duration
	creationGracePeriod time.Duration

	// The below structs embed the set of interfaces used to implement the
	// managed resource reconciler. We do this primarily for readability, so
	// that the reconciler logic reads r.external.Connect(),
	// r.managed.Delete(), etc.
	external mrExternal
	managed  mrManaged

	log    logging.Logger
	record event.Recorder
}

type mrManaged struct {
	CriticalAnnotationUpdater
	resource.Finalizer
}

func defaultMRManaged(m manager.Manager) mrManaged {
	return mrManaged{
		CriticalAnnotationUpdater: NewRetryingCriticalAnnotationUpdater(m.GetClient()),
		Finalizer:                 resource.NewAPIFinalizer(m.GetClient(), FinalizerName),
	}
}

type mrExternal struct {
	ExternalConnectDisconnecter
}

func defaultMRExternal() mrExternal {
	return mrExternal{
		ExternalConnectDisconnecter: NewNopDisconnecter(&NopConnecter{}),
	}
}

// PollIntervalHook represents the function type passed to the
// WithPollIntervalHook option to support dynamic computation of the poll
// interval.
type PollIntervalHook func(managed resource.Managed, pollInterval time.Duration) time.Duration

func defaultPollIntervalHook(_ resource.Managed, pollInterval time.Duration) time.Duration {
	return pollInterval
}

// A ReconcilerOption configures a Reconciler.
type ReconcilerOption func(*Reconciler)

// WithTimeout specifies the timeout duration cumulatively for all the calls happen
// in the reconciliation function. In case the deadline exceeds, reconciler will
// still have some time to make the necessary calls to report the error such as
// status update.
func WithTimeout(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.timeout = duration
	}
}

// WithPollInterval specifies how long the Reconciler should wait before queueing
// a new reconciliation after a successful reconcile. The Reconciler requeues
// after a specified duration when it is not actively waiting for an external
// operation, but wishes to check whether an existing external resource needs to
// be synced to its Managed resource.
func WithPollInterval(after time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.pollInterval = after
	}
}

// WithPollIntervalHook adds a hook that can be used to configure the
// delay before an up-to-date resource is reconciled again after a successful
// reconcile. If this option is passed multiple times, only the latest hook
// will be used.
func WithPollIntervalHook(hook PollIntervalHook) ReconcilerOption {
	return func(r *Reconciler) {
		r.pollIntervalHook = hook
	}
}

// WithPollJitterHook adds a simple PollIntervalHook to add jitter to the poll
// interval used when queuing a new reconciliation after a successful
// reconcile. The added jitter will be a random duration between -jitter and
// +jitter. This option wraps WithPollIntervalHook, and is subject to the same
// constraint that only the latest hook will be used.
func WithPollJitterHook(jitter time.Duration) ReconcilerOption {
	return WithPollIntervalHook(func(_ resource.Managed, pollInterval time.Duration) time.Duration {
		return pollInterval + time.Duration((rand.Float64()-0.5)*2*float64(jitter)) //nolint:gosec // No need for secure randomness.
	})
}

// WithCreationGracePeriod configures an optional period during which we will
// wait for the external API to report that a newly created external resource
// exists. This allows us to tolerate eventually consistent APIs that do not
// immediately report that newly created resources exist when queried. All
// resources have a 30 second grace period by default.
func WithCreationGracePeriod(d time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.creationGracePeriod = d
	}
}

// WithExternalConnecter specifies how the Reconciler should connect to the API
// used to sync and delete external resources.
func WithExternalConnecter(c ExternalConnecter) ReconcilerOption {
	return func(r *Reconciler) {
		r.external.ExternalConnectDisconnecter = NewNopDisconnecter(c)
	}
}

// WithExternalConnectDisconnecter specifies how the Reconciler should connect and disconnect to the API
// used to sync and delete external resources.
func WithExternalConnectDisconnecter(c ExternalConnectDisconnecter) ReconcilerOption {
	return func(r *Reconciler) {
		r.external.ExternalConnectDisconnecter = c
	}
}

// WithCriticalAnnotationUpdater specifies how the Reconciler should update a
// managed resource's critical annotations. Implementations typically contain
// some kind of retry logic to increase the likelihood that critical annotations
// (like non-deterministic external names) will be persisted.
func WithCriticalAnnotationUpdater(u CriticalAnnotationUpdater) ReconcilerOption {
	return func(r *Reconciler) {
		r.managed.CriticalAnnotationUpdater = u
	}
}

// WithFinalizer specifies how the Reconciler should add and remove
// finalizers to and from the managed resource.
func WithFinalizer(f resource.Finalizer) ReconcilerOption {
	return func(r *Reconciler) {
		r.managed.Finalizer = f
	}
}

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(l logging.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = l
	}
}

// WithRecorder specifies how the Reconciler should record events.
func WithRecorder(er event.Recorder) ReconcilerOption {
	return func(r *Reconciler) {
		r.record = er
	}
}

// NewReconciler returns a Reconciler that reconciles managed resources of the
// supplied ManagedKind with resources in an external system such as a cloud
// provider API. It panics if asked to reconcile a managed resource kind that is
// not registered with the supplied manager's runtime.Scheme. The returned
// Reconciler reconciles with a dummy, no-op 'external system' by default;
// callers should supply an ExternalConnector that returns an ExternalClient
// capable of managing resources in a real system.
func NewReconciler(m manager.Manager, of resource.ManagedKind, o ...ReconcilerOption) *Reconciler {
	nm := func() resource.Managed {
		return resource.MustCreateObject(schema.GroupVersionKind(of), m.GetScheme()).(resource.Managed)
	}

	// Panic early if we've been asked to reconcile a resource kind that has not
	// been registered with our controller manager's scheme.
	_ = nm()

	r := &Reconciler{
		client:              m.GetClient(),
		newManaged:          nm,
		pollInterval:        defaultpollInterval,
		pollIntervalHook:    defaultPollIntervalHook,
		creationGracePeriod: defaultGracePeriod,
		timeout:             reconcileTimeout,
		managed:             defaultMRManaged(m),
		external:            defaultMRExternal(),
		log:                 logging.NewNopLogger(),
		record:              event.NewNopRecorder(),
	}

	for _, ro := range o {
		ro(r)
	}

	return r
}

// Reconcile a managed resource with an external resource.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { //nolint:gocyclo // See note below.
	log := r.log.WithValues("request", req)
	log.Debug("Reconciling")

	ctx, cancel := context.WithTimeout(ctx, r.timeout+reconcileGracePeriod)
	defer cancel()

	externalCtx, externalCancel := context.WithTimeout(ctx, r.timeout)
	defer externalCancel()

	managed := r.newManaged()
	if err := r.client.Get(ctx, req.NamespacedName, managed); err != nil {
		// There's no need to requeue if we no longer exist. Otherwise we'll be
		// requeued implicitly because we return an error.
		log.Debug("Cannot get managed resource", "error", err)
		return reconcile.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetManaged)
	}

	record := r.record.WithAnnotations("external-name", meta.GetExternalName(managed))
	log = log.WithValues(
		"uid", managed.GetUID(),
		"version", managed.GetResourceVersion(),
		"external-name", meta.GetExternalName(managed),
	)

	// Check the pause annotation and return if it has the value "true"
	// after logging, publishing an event and updating the SYNC status condition
	if meta.IsPaused(managed) {
		log.Debug("Reconciliation is paused via the pause annotation", "annotation", meta.AnnotationKeyReconciliationPaused, "value", "true")
		record.Event(managed, event.Normal(reasonReconciliationPaused, "Reconciliation is paused via the pause annotation"))
		managed.SetConditions(prv1.ReconcilePaused())
		// if the pause annotation is removed, we will have a chance to reconcile again and resume
		// and if status update fails, we will reconcile again to retry to update the status
		return reconcile.Result{}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// If managed resource has a deletion timestamp and and a deletion policy of
	// Orphan, we do not need to observe the external resource before attempting
	// to remove finalizer.
	if meta.WasDeleted(managed) && !meta.ShouldDelete(managed) {
		log = log.WithValues("deletion-timestamp", managed.GetDeletionTimestamp())

		if err := r.managed.RemoveFinalizer(ctx, managed); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			managed.SetConditions(prv1.Deleting(), prv1.ReconcileError(err))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We've successfully unpublished our managed resource's connection
		// details and removed our finalizer. If we assume we were the only
		// controller that added a finalizer to this resource then it should no
		// longer exist and thus there is no point trying to update its status.
		log.Debug("Successfully deleted managed resource")
		return reconcile.Result{Requeue: false}, nil
	}

	// If we started but never completed creation of an external resource we
	// may have lost critical information. For example if we didn't persist
	// an updated external name we've leaked a resource. The safest thing to
	// do is to refuse to proceed.
	if meta.ExternalCreateIncomplete(managed) {
		log.Debug(errCreateIncomplete)
		record.Event(managed, event.Warning(reasonCannotInitialize, errors.New(errCreateIncomplete)))
		managed.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.New(errCreateIncomplete)))
		return reconcile.Result{Requeue: false}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	external, err := r.external.Connect(externalCtx, managed)
	if err != nil {
		// We'll usually hit this case if our Provider or its secret are missing
		// or invalid. If this is first time we encounter this issue we'll be
		// requeued implicitly when we update our status with the new error
		// condition. If not, we requeue explicitly, which will trigger
		// backoff.
		log.Debug("Cannot connect to provider", "error", err)
		if kerrors.IsConflict(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		record.Event(managed, event.Warning(reasonCannotConnect, err))
		managed.SetConditions(prv1.ReconcileError(errors.Wrap(err, errReconcileConnect)))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}
	defer func() {
		if err := r.external.Disconnect(ctx); err != nil {
			log.Debug("Cannot disconnect from provider", "error", err)
			record.Event(managed, event.Warning(reasonCannotDisconnect, err))
		}
	}()

	observation, err := external.Observe(externalCtx, managed)
	if err != nil {
		// We'll usually hit this case if our Provider credentials are invalid
		// or insufficient for observing the external resource type we're
		// concerned with. If this is the first time we encounter this issue
		// we'll be requeued implicitly when we update our status with the new
		// error condition. If not, we requeue explicitly, which will
		// trigger backoff.
		log.Debug("Cannot observe external resource", "error", err)
		if kerrors.IsConflict(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		record.Event(managed, event.Warning(reasonCannotObserve, err))
		managed.SetConditions(prv1.ReconcileError(errors.Wrap(err, errReconcileObserve)))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// In the observe-only mode, !observation.ResourceExists will be an error
	// case, and we will explicitly return this information to the user.
	if !observation.ResourceExists && meta.ShouldOnlyObserve(managed) {
		record.Event(managed, event.Warning(reasonCannotObserve, errors.New(errExternalResourceNotExist)))
		managed.SetConditions(prv1.ReconcileError(errors.Wrap(errors.New(errExternalResourceNotExist), errReconcileObserve)))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// If this resource has a non-zero creation grace period we want to wait
	// for that period to expire before we trust that the resource really
	// doesn't exist. This is because some external APIs are eventually
	// consistent and may report that a recently created resource does not
	// exist.
	if !observation.ResourceExists && meta.ExternalCreateSucceededDuring(managed, r.creationGracePeriod) {
		log.Debug("Waiting for external resource existence to be confirmed")
		record.Event(managed, event.Normal(reasonPending, "Waiting for external resource existence to be confirmed"))
		return reconcile.Result{Requeue: true}, nil
	}

	if meta.WasDeleted(managed) {
		log = log.WithValues("deletion-timestamp", managed.GetDeletionTimestamp())

		// We'll only reach this point if deletion policy is not orphan, so we
		// are safe to call external deletion if external resource exists.
		if observation.ResourceExists && meta.ShouldDelete(managed) {
			if err := external.Delete(externalCtx, managed); err != nil {
				// We'll hit this condition if we can't delete our external
				// resource, for example if our provider credentials don't have
				// access to delete it. If this is the first time we encounter
				// this issue we'll be requeued implicitly when we update our
				// status with the new error condition. If not, we want requeue
				// explicitly, which will trigger backoff.
				log.Debug("Cannot delete external resource", "error", err)
				record.Event(managed, event.Warning(reasonCannotDelete, err))
				managed.SetConditions(prv1.Deleting(), prv1.ReconcileError(errors.Wrap(err, errReconcileDelete)))
				return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
			}

			// We've successfully requested deletion of our external resource.
			// We queue another reconcile after a short wait rather than
			// immediately finalizing our delete in order to verify that the
			// external resource was actually deleted. If it no longer exists
			// we'll skip this block on the next reconcile and proceed to
			// unpublish and finalize. If it still exists we'll re-enter this
			// block and try again.
			log.Debug("Successfully requested deletion of external resource")
			record.Event(managed, event.Normal(reasonDeleted, "Successfully requested deletion of external resource"))
			managed.SetConditions(prv1.Deleting(), prv1.ReconcileSuccess())
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		if err := r.managed.RemoveFinalizer(ctx, managed); err != nil {
			// If this is the first time we encounter this issue we'll be
			// requeued implicitly when we update our status with the new error
			// condition. If not, we requeue explicitly, which will trigger
			// backoff.
			log.Debug("Cannot remove managed resource finalizer", "error", err)
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			managed.SetConditions(prv1.Deleting(), prv1.ReconcileError(err))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We've successfully deleted our external resource (if necessary) and
		// removed our finalizer. If we assume we were the only controller that
		// added a finalizer to this resource then it should no longer exist and
		// thus there is no point trying to update its status.
		log.Debug("Successfully deleted managed resource")
		return reconcile.Result{Requeue: false}, nil
	}

	if err := r.managed.AddFinalizer(ctx, managed); err != nil {
		// If this is the first time we encounter this issue we'll be requeued
		// implicitly when we update our status with the new error condition. If
		// not, we requeue explicitly, which will trigger backoff.
		log.Debug("Cannot add finalizer", "error", err)
		if kerrors.IsConflict(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		managed.SetConditions(prv1.ReconcileError(err))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if !observation.ResourceExists && meta.ShouldCreate(managed) {
		// We write this annotation for two reasons. Firstly, it helps
		// us to detect the case in which we fail to persist critical
		// information (like the external name) that may be set by the
		// subsequent external.Create call. Secondly, it guarantees that
		// we're operating on the latest version of our resource. We
		// don't use the CriticalAnnotationUpdater because we _want_ the
		// update to fail if we get a 409 due to a stale version.
		meta.SetExternalCreatePending(managed, time.Now())
		if err := r.client.Update(ctx, managed); err != nil {
			log.Debug(errUpdateManaged, "error", err)
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			record.Event(managed, event.Warning(reasonCannotUpdateManaged, errors.Wrap(err, errUpdateManaged)))
			managed.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.Wrap(err, errUpdateManaged)))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		err = external.Create(externalCtx, managed)
		if err != nil {
			// We'll hit this condition if we can't create our external
			// resource, for example if our provider credentials don't have
			// access to create it. If this is the first time we encounter this
			// issue we'll be requeued implicitly when we update our status with
			// the new error condition. If not, we requeue explicitly, which will trigger backoff.
			log.Debug("Cannot create external resource", "error", err)
			if kerrors.IsConflict(err) {
				return reconcile.Result{Requeue: true}, nil
			}
			record.Event(managed, event.Warning(reasonCannotCreate, err))

			// We handle annotations specially here because it's
			// critical that they are persisted to the API server.
			// If we don't add the external-create-failed annotation
			// the reconciler will refuse to proceed, because it
			// won't know whether or not it created an external
			// resource.
			meta.SetExternalCreateFailed(managed, time.Now())
			if err := r.managed.UpdateCriticalAnnotations(ctx, managed); err != nil {
				log.Debug(errUpdateManagedAnnotations, "error", err)
				record.Event(managed, event.Warning(reasonCannotUpdateManaged, errors.Wrap(err, errUpdateManagedAnnotations)))

				// We only log and emit an event here rather
				// than setting a status condition and returning
				// early because presumably it's more useful to
				// set our status condition to the reason the
				// create failed.
			}

			managed.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.Wrap(err, errReconcileCreate)))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// In some cases our external-name may be set by Create above.
		log = log.WithValues("external-name", meta.GetExternalName(managed))
		record = r.record.WithAnnotations("external-name", meta.GetExternalName(managed))

		// We handle annotations specially here because it's critical
		// that they are persisted to the API server. If we don't remove
		// add the external-create-succeeded annotation the reconciler
		// will refuse to proceed, because it won't know whether or not
		// it created an external resource. This is also important in
		// cases where we must record an external-name annotation set by
		// the Create call. Any other changes made during Create will be
		// reverted when annotations are updated; at the time of writing
		// Create implementations are advised not to alter status, but
		// we may revisit this in future.
		meta.SetExternalCreateSucceeded(managed, time.Now())
		if err := r.managed.UpdateCriticalAnnotations(ctx, managed); err != nil {
			log.Debug(errUpdateManagedAnnotations, "error", err)
			record.Event(managed, event.Warning(reasonCannotUpdateManaged, errors.Wrap(err, errUpdateManagedAnnotations)))
			managed.SetConditions(prv1.Creating(), prv1.ReconcileError(errors.Wrap(err, errUpdateManagedAnnotations)))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}

		// We've successfully created our external resource. In many cases the
		// creation process takes a little time to finish. We requeue explicitly
		// order to observe the external resource to determine whether it's
		// ready for use.
		log.Debug("Successfully requested creation of external resource")
		record.Event(managed, event.Normal(reasonCreated, "Successfully requested creation of external resource"))
		managed.SetConditions(prv1.Creating(), prv1.ReconcileSuccess())
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if observation.ResourceLateInitialized {
		// Note that this update may reset any pending updates to the status of
		// the managed resource from when it was observed above. This is because
		// the API server replies to the update with its unchanged view of the
		// resource's status, which is subsequently deserialized into managed.
		// This is usually tolerable because the update will implicitly requeue
		// an immediate reconcile which should re-observe the external resource
		// and persist its status.
		if err := r.client.Update(ctx, managed); err != nil {
			log.Debug(errUpdateManaged, "error", err)
			record.Event(managed, event.Warning(reasonCannotUpdateManaged, err))
			managed.SetConditions(prv1.ReconcileError(errors.Wrap(err, errUpdateManaged)))
			return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
		}
	}

	if observation.ResourceUpToDate {
		// We did not need to create, update, or delete our external resource.
		// Per the below issue nothing will notify us if and when the external
		// resource we manage changes, so we requeue a speculative reconcile
		// after the specified poll interval in order to observe it and react
		// accordingly.
		// https://github.com/crossplane/crossplane/issues/289
		reconcileAfter := r.pollIntervalHook(managed, r.pollInterval)
		log.Debug("External resource is up to date", "requeue-after", time.Now().Add(reconcileAfter))
		managed.SetConditions(prv1.ReconcileSuccess())
		return reconcile.Result{RequeueAfter: reconcileAfter}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	if observation.Diff != "" {
		log.Debug("External resource differs from desired state", "diff", observation.Diff)
	}

	// skip the update if the management policy is set to ignore updates
	if !meta.ShouldUpdate(managed) {
		reconcileAfter := r.pollIntervalHook(managed, r.pollInterval)
		log.Debug("Skipping update due to managementPolicies. Reconciliation succeeded", "requeue-after", time.Now().Add(reconcileAfter))
		managed.SetConditions(prv1.ReconcileSuccess())
		return reconcile.Result{RequeueAfter: reconcileAfter}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	err = external.Update(externalCtx, managed)
	if err != nil {
		// We'll hit this condition if we can't update our external resource,
		// for example if our provider credentials don't have access to update
		// it. If this is the first time we encounter this issue we'll be
		// requeued implicitly when we update our status with the new error
		// condition. If not, we requeue explicitly, which will trigger backoff.
		log.Debug("Cannot update external resource")
		record.Event(managed, event.Warning(reasonCannotUpdate, err))
		managed.SetConditions(prv1.ReconcileError(errors.Wrap(err, errReconcileUpdate)))
		return reconcile.Result{Requeue: true}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
	}

	// We've successfully updated our external resource. Per the below issue
	// nothing will notify us if and when the external resource we manage
	// changes, so we requeue a speculative reconcile after the specified poll
	// interval in order to observe it and react accordingly.
	// https://github.com/crossplane/crossplane/issues/289
	reconcileAfter := r.pollIntervalHook(managed, r.pollInterval)
	log.Debug("Successfully requested update of external resource", "requeue-after", time.Now().Add(reconcileAfter))
	record.Event(managed, event.Normal(reasonUpdated, "Successfully requested update of external resource"))
	managed.SetConditions(prv1.ReconcileSuccess())
	return reconcile.Result{RequeueAfter: reconcileAfter}, errors.Wrap(r.client.Status().Update(ctx, managed), errUpdateManagedStatus)
}
