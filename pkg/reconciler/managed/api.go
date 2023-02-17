package managed

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/krateoplatformops/provider-runtime/pkg/meta"
	"github.com/krateoplatformops/provider-runtime/pkg/resource"
)

// Error strings.
const (
	errCreateOrUpdateSecret      = "cannot create or update connection secret"
	errUpdateManaged             = "cannot update managed resource"
	errUpdateManagedStatus       = "cannot update managed resource status"
	errResolveReferences         = "cannot resolve references"
	errUpdateCriticalAnnotations = "cannot update critical annotations"
)

type NoopInitializer struct{ client client.Client }

// NoopInitializer returns a new NoopInitializer.
func NewNoopInitializer(c client.Client) *NoopInitializer {
	return &NoopInitializer{client: c}
}

// Initialize the given managed resource.
func (a *NoopInitializer) Initialize(ctx context.Context, mg resource.Managed) error {
	return nil
}

/*
// NameAsExternalName writes the name of the managed resource to
// the external name annotation field in order to be used as name of
// the external resource in provider.
type NameAsExternalName struct{ client client.Client }

// NewNameAsExternalName returns a new NameAsExternalName.
func NewNameAsExternalName(c client.Client) *NameAsExternalName {
	return &NameAsExternalName{client: c}
}

// Initialize the given managed resource.
func (a *NameAsExternalName) Initialize(ctx context.Context, mg resource.Managed) error {
	if meta.GetExternalName(mg) != "" {
		return nil
	}
	meta.SetExternalName(mg, mg.GetName())
	return errors.Wrap(a.client.Update(ctx, mg), errUpdateManaged)
}

*/

// An APISimpleReferenceResolver resolves references from one managed resource
// to others by calling the referencing resource's ResolveReferences method, if
// any.
type APISimpleReferenceResolver struct {
	client client.Client
}

// NewAPISimpleReferenceResolver returns a ReferenceResolver that resolves
// references from one managed resource to others by calling the referencing
// resource's ResolveReferences method, if any.
func NewAPISimpleReferenceResolver(c client.Client) *APISimpleReferenceResolver {
	return &APISimpleReferenceResolver{client: c}
}

// ResolveReferences of the supplied managed resource by calling its
// ResolveReferences method, if any.
func (a *APISimpleReferenceResolver) ResolveReferences(ctx context.Context, mg resource.Managed) error {
	rr, ok := mg.(interface {
		ResolveReferences(context.Context, client.Reader) error
	})
	if !ok {
		// This managed resource doesn't have any references to resolve.
		return nil
	}

	existing := mg.DeepCopyObject()
	if err := rr.ResolveReferences(ctx, a.client); err != nil {
		return errors.Wrap(err, errResolveReferences)
	}

	if cmp.Equal(existing, mg) {
		// The resource didn't change during reference resolution.
		return nil
	}

	return errors.Wrap(a.client.Update(ctx, mg), errUpdateManaged)
}

// A RetryingCriticalAnnotationUpdater is a CriticalAnnotationUpdater that
// retries annotation updates in the face of API server errors.
type RetryingCriticalAnnotationUpdater struct {
	client client.Client
}

// NewRetryingCriticalAnnotationUpdater returns a CriticalAnnotationUpdater that
// retries annotation updates in the face of API server errors.
func NewRetryingCriticalAnnotationUpdater(c client.Client) *RetryingCriticalAnnotationUpdater {
	return &RetryingCriticalAnnotationUpdater{client: c}
}

// UpdateCriticalAnnotations updates (i.e. persists) the annotations of the
// supplied Object. It retries in the face of any API server error several times
// in order to ensure annotations that contain critical state are persisted. Any
// pending changes to the supplied Object's spec, status, or other metadata are
// reset to their current state according to the API server.
func (u *RetryingCriticalAnnotationUpdater) UpdateCriticalAnnotations(ctx context.Context, o client.Object) error {
	a := o.GetAnnotations()
	err := retry.OnError(retry.DefaultRetry, resource.IsAPIError, func() error {
		nn := types.NamespacedName{Name: o.GetName()}
		if err := u.client.Get(ctx, nn, o); err != nil {
			return err
		}
		meta.AddAnnotations(o, a)
		return u.client.Update(ctx, o)
	})
	return errors.Wrap(err, errUpdateCriticalAnnotations)
}
