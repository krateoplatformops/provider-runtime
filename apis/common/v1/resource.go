package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// CredentialSelectors provides selectors for extracting credentials.
type CredentialSelectors struct {
	// Env is a reference to an environment variable that contains credentials
	// that must be used to connect to the provider.
	// +optional
	Env *EnvSelector `json:"env,omitempty"`

	// A SecretRef is a reference to a secret key that contains the credentials
	// that must be used to connect to the provider.
	// +optional
	SecretRef *SecretKeySelector `json:"secretRef,omitempty"`
}

// EnvSelector selects an environment variable.
type EnvSelector struct {
	// Name is the name of an environment variable.
	Name string `json:"name"`
}

// A SecretReference is a reference to a secret in an arbitrary namespace.
type SecretReference struct {
	// Name of the secret.
	Name string `json:"name"`

	// Namespace of the secret.
	Namespace string `json:"namespace"`
}

// A SecretKeySelector is a reference to a secret key in an arbitrary namespace.
type SecretKeySelector struct {
	SecretReference `json:",inline"`

	// The key to select.
	Key string `json:"key"`
}

// ManagedStatus represents the observed state of a managed resource.
type ManagedStatus struct {
	ConditionedStatus `json:",inline"`
}

// A ManagedSpec defines the desired state of a managed resource.
type ManagedSpec struct {
	// ProviderConfigReference specifies how the provider that will be used to
	// create, observe, update, and delete this managed resource should be
	// configured.
	// +kubebuilder:default={"name": "default"}
	ProviderConfigReference *Reference `json:"providerConfigRef,omitempty"`

	// DeletionPolicy specifies what will happen to the underlying external
	// when this managed resource is deleted - either "Delete" or "Orphan" the
	// external resource.
	// +optional
	// +kubebuilder:default=Delete
	DeletionPolicy DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// Policy represents the Resolve and Resolution policies of Reference instance.
type Policy struct {
	// Resolve specifies when this reference should be resolved. The default
	// is 'IfNotPresent', which will attempt to resolve the reference only when
	// the corresponding field is not present. Use 'Always' to resolve the
	// reference on every reconcile.
	// +optional
	// +kubebuilder:validation:Enum=Always;IfNotPresent
	Resolve *ResolvePolicy `json:"resolve,omitempty"`

	// Resolution specifies whether resolution of this reference is required.
	// The default is 'Required', which means the reconcile will fail if the
	// reference cannot be resolved. 'Optional' means this reference will be
	// a no-op if it cannot be resolved.
	// +optional
	// +kubebuilder:default=Required
	// +kubebuilder:validation:Enum=Required;Optional
	Resolution *ResolutionPolicy `json:"resolution,omitempty"`
}

// IsResolutionPolicyOptional checks whether the resolution policy of relevant reference is Optional.
func (p *Policy) IsResolutionPolicyOptional() bool {
	if p == nil || p.Resolution == nil {
		return false
	}
	return *p.Resolution == ResolutionPolicyOptional
}

// IsResolvePolicyAlways checks whether the resolution policy of relevant reference is Always.
func (p *Policy) IsResolvePolicyAlways() bool {
	if p == nil || p.Resolve == nil {
		return false
	}
	return *p.Resolve == ResolvePolicyAlways
}

// A Reference to a named object.
type Reference struct {
	// Name of the referenced object.
	Name string `json:"name"`

	// Policies for referencing.
	// +optional
	Policy *Policy `json:"policy,omitempty"`
}

// namespace is already known.
type TypedReference struct {
	// APIVersion of the referenced object.
	APIVersion string `json:"apiVersion"`

	// Kind of the referenced object.
	Kind string `json:"kind"`

	// Name of the referenced object.
	Name string `json:"name"`

	// UID of the referenced object.
	// +optional
	UID types.UID `json:"uid,omitempty"`
}

// SetGroupVersionKind sets the Kind and APIVersion of a TypedReference.
func (obj *TypedReference) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	obj.APIVersion, obj.Kind = gvk.ToAPIVersionAndKind()
}

// GroupVersionKind gets the GroupVersionKind of a TypedReference.
func (obj *TypedReference) GroupVersionKind() schema.GroupVersionKind {
	return schema.FromAPIVersionAndKind(obj.APIVersion, obj.Kind)
}

// GetObjectKind get the ObjectKind of a TypedReference.
func (obj *TypedReference) GetObjectKind() schema.ObjectKind { return obj }
