package resource

import (
	"context"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// An Object is a Kubernetes object.
type Object interface {
	metav1.Object
	runtime.Object
}

// An Orphanable resource may specify a DeletionPolicy.
type Orphanable interface {
	SetDeletionPolicy(p prv1.DeletionPolicy)
	GetDeletionPolicy() prv1.DeletionPolicy
}

// A ProviderReferencer may reference a provider resource.
type ProviderReferencer interface {
	GetProviderReference() *prv1.Reference
	SetProviderReference(p *prv1.Reference)
}

// A ProviderConfigReferencer may reference a provider config resource.
type ProviderConfigReferencer interface {
	GetProviderConfigReference() *prv1.Reference
	SetProviderConfigReference(p *prv1.Reference)
}

// A Managed is a Kubernetes object representing a concrete managed
// resource (e.g. a CloudSQL instance).
type Managed interface {
	Object
	ProviderReferencer
	ProviderConfigReferencer
	Orphanable
	Conditioned
}

// A Conditioned may have conditions set or retrieved. Conditions are typically
// indicate the status of both a resource and its reconciliation process.
type Conditioned interface {
	SetConditions(c ...prv1.Condition)
	GetCondition(prv1.ConditionType) prv1.Condition
}

// A Finalizer manages the finalizers on the resource.
type Finalizer interface {
	AddFinalizer(ctx context.Context, obj Object) error
	RemoveFinalizer(ctx context.Context, obj Object) error
}
