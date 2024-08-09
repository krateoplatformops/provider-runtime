package v1

// A Reference to a named object.
type Reference struct {
	// Name of the referenced object.
	Name string `json:"name"`

	// Namespace of the referenced object.
	Namespace string `json:"namespace"`
}

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

// A SecretKeySelector is a reference to a secret key in an arbitrary namespace.
type SecretKeySelector struct {
	Reference `json:",inline"`

	// The key to select.
	Key string `json:"key"`
}

// A ConfigMapKeySelector is a reference to a configmap key in an arbitrary namespace.
type ConfigMapKeySelector struct {
	Reference `json:",inline"`

	// The key to select.
	Key string `json:"key"`
}

// ManagedStatus represents the observed state of a managed resource.
// type ManagedStatus struct {
// 	ConditionedStatus `json:",inline"`
// }
