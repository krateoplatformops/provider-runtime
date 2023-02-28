// Package meta contains functions for dealing with Kubernetes object metadata.
package meta

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AnnotationKeyExternalName is the key in the annotations map of a
	// resource for the name of the resource as it appears on provider's
	// systems.
	AnnotationKeyExternalName = "krateo.io/external-name"

	// AnnotationKeyExternalCreatePending is the key in the annotations map
	// of a resource that indicates the last time creation of the external
	// resource was pending (i.e. about to happen). Its value must be an
	// RFC3999 timestamp.
	AnnotationKeyExternalCreatePending = "krateo.io/external-create-pending"

	// AnnotationKeyExternalCreateSucceeded is the key in the annotations
	// map of a resource that represents the last time the external resource
	// was created successfully. Its value must be an RFC3339 timestamp,
	// which can be used to determine how long ago a resource was created.
	// This is useful for eventually consistent APIs that may take some time
	// before the API called by Observe will report that a recently created
	// external resource exists.
	AnnotationKeyExternalCreateSucceeded = "krateo.io/external-create-succeeded"

	// AnnotationKeyExternalCreateFailed is the key in the annotations map
	// of a resource that indicates the last time creation of the external
	// resource failed. Its value must be an RFC3999 timestamp.
	AnnotationKeyExternalCreateFailed = "krateo.io/external-create-failed"

	// AnnotationKeyReconciliationPaused is the key in the annotations map
	// of a resource that indicates that further reconciliations on the
	// resource are paused. All create/update/delete/generic events on
	// the resource will be filtered and thus no further reconcile requests
	// will be queued for the resource.
	AnnotationKeyReconciliationPaused = "krateo.io/paused"

	// AnnotationKeyConnectorVerbose is the key in the annotations map
	// of a resource that indicates that the external client has verbose info enabled.
	AnnotationKeyConnectorVerbose = "krateo.io/connector-verbose"

	// AnnotationKeyManagementPolicy is the key in the annotations map
	// of a resource to instruct the provider to manage resources in a fine-grained way.
	// default: The provider can fully manage the resource.
	//          This is the default policy.
	// observe-create-update: The provider can observe, create, or update the
	//                        resource, but can not delete it.
	// observe-delete: The provider can observe or delete the resource, but
	//                 can not create and update it.
	// observe: The provider can only observe the resource.
	//          This maps to the read-only scenario where the resource is fully controlled by third party application.
	AnnotationKeyManagementPolicy = "krateo.io/management-policy"
)

const (
	// ManagementPolicyDefault means the provider can fully manage the resource.
	ManagementPolicyDefault = "default"
	// ManagementPolicyObserveCreateUpdate means the provider can observe, create,
	// or update the resource, but can not delete it.
	ManagementPolicyObserveCreateUpdate = "observe-create-update"
	// ManagementPolicyObserveDelete means the provider can observe or delete
	// the resource, but can not create and update it.
	ManagementPolicyObserveDelete = "observe-delete"
	// ManagementPolicyObserve means the provider can only observe the resource.
	ManagementPolicyObserve = "observe"

	// ActionCreate means to create an Object
	ActionCreate = "create"
	// ActionUpdate means to update an Object
	ActionUpdate = "update"
	// ActionDelete means to delete an Object
	ActionDelete = "delete"
)

// AddFinalizer to the supplied Kubernetes object's metadata.
func AddFinalizer(o metav1.Object, finalizer string) {
	f := o.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return
		}
	}
	o.SetFinalizers(append(f, finalizer))
}

// RemoveFinalizer from the supplied Kubernetes object's metadata.
func RemoveFinalizer(o metav1.Object, finalizer string) {
	f := o.GetFinalizers()
	for i, e := range f {
		if e == finalizer {
			f = append(f[:i], f[i+1:]...)
		}
	}
	o.SetFinalizers(f)
}

// FinalizerExists checks whether given finalizer is already set.
func FinalizerExists(o metav1.Object, finalizer string) bool {
	f := o.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}

// AddLabels to the supplied object.
func AddLabels(o metav1.Object, labels map[string]string) {
	l := o.GetLabels()
	if l == nil {
		o.SetLabels(labels)
		return
	}
	for k, v := range labels {
		l[k] = v
	}
	o.SetLabels(l)
}

// RemoveLabels with the supplied keys from the supplied object.
func RemoveLabels(o metav1.Object, labels ...string) {
	l := o.GetLabels()
	if l == nil {
		return
	}
	for _, k := range labels {
		delete(l, k)
	}
	o.SetLabels(l)
}

// AddAnnotations to the supplied object.
func AddAnnotations(o metav1.Object, annotations map[string]string) {
	a := o.GetAnnotations()
	if a == nil {
		o.SetAnnotations(annotations)
		return
	}
	for k, v := range annotations {
		a[k] = v
	}
	o.SetAnnotations(a)
}

// RemoveAnnotations with the supplied keys from the supplied object.
func RemoveAnnotations(o metav1.Object, annotations ...string) {
	a := o.GetAnnotations()
	if a == nil {
		return
	}
	for _, k := range annotations {
		delete(a, k)
	}
	o.SetAnnotations(a)
}

// WasDeleted returns true if the supplied object was deleted from the API server.
func WasDeleted(o metav1.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

// WasCreated returns true if the supplied object was created in the API server.
func WasCreated(o metav1.Object) bool {
	// This looks a little different from WasDeleted because DeletionTimestamp
	// returns a reference while CreationTimestamp returns a value.
	t := o.GetCreationTimestamp()
	return !t.IsZero()
}

// GetExternalName returns the external name annotation value on the resource.
func GetExternalName(o metav1.Object) string {
	return o.GetAnnotations()[AnnotationKeyExternalName]
}

// SetExternalName sets the external name annotation of the resource.
func SetExternalName(o metav1.Object, name string) {
	AddAnnotations(o, map[string]string{AnnotationKeyExternalName: name})
}

// GetExternalCreatePending returns the time at which the external resource
// was most recently pending creation.
func GetExternalCreatePending(o metav1.Object) time.Time {
	a := o.GetAnnotations()[AnnotationKeyExternalCreatePending]
	t, err := time.Parse(time.RFC3339, a)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SetExternalCreatePending sets the time at which the external resource was
// most recently pending creation to the supplied time.
func SetExternalCreatePending(o metav1.Object, t time.Time) {
	AddAnnotations(o, map[string]string{AnnotationKeyExternalCreatePending: t.Format(time.RFC3339)})
}

// GetExternalCreateSucceeded returns the time at which the external resource
// was most recently created.
func GetExternalCreateSucceeded(o metav1.Object) time.Time {
	a := o.GetAnnotations()[AnnotationKeyExternalCreateSucceeded]
	t, err := time.Parse(time.RFC3339, a)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SetExternalCreateSucceeded sets the time at which the external resource was
// most recently created to the supplied time.
func SetExternalCreateSucceeded(o metav1.Object, t time.Time) {
	AddAnnotations(o, map[string]string{AnnotationKeyExternalCreateSucceeded: t.Format(time.RFC3339)})
}

// GetExternalCreateFailed returns the time at which the external resource
// recently failed to create.
func GetExternalCreateFailed(o metav1.Object) time.Time {
	a := o.GetAnnotations()[AnnotationKeyExternalCreateFailed]
	t, err := time.Parse(time.RFC3339, a)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SetExternalCreateFailed sets the time at which the external resource most
// recently failed to create.
func SetExternalCreateFailed(o metav1.Object, t time.Time) {
	AddAnnotations(o, map[string]string{AnnotationKeyExternalCreateFailed: t.Format(time.RFC3339)})
}

// ExternalCreateIncomplete returns true if creation of the external resource
// appears to be incomplete. We deem creation to be incomplete if the 'external
// create pending' annotation is the newest of all tracking annotations that are
// set (i.e. pending, succeeded, and failed).
func ExternalCreateIncomplete(o metav1.Object) bool {
	pending := GetExternalCreatePending(o)
	succeeded := GetExternalCreateSucceeded(o)
	failed := GetExternalCreateFailed(o)

	// If creation never started it can't be incomplete.
	if pending.IsZero() {
		return false
	}

	latest := succeeded
	if failed.After(succeeded) {
		latest = failed
	}

	return pending.After(latest)
}

// ExternalCreateSucceededDuring returns true if creation of the external
// resource that corresponds to the supplied managed resource succeeded within
// the supplied duration.
func ExternalCreateSucceededDuring(o metav1.Object, d time.Duration) bool {
	t := GetExternalCreateSucceeded(o)
	if t.IsZero() {
		return false
	}
	return time.Since(t) < d
}

// IsPaused returns true if the object has the AnnotationKeyReconciliationPaused
// annotation set to `true`.
func IsPaused(o metav1.Object) bool {
	return o.GetAnnotations()[AnnotationKeyReconciliationPaused] == "true"
}

// IsVerbose returns true if the object has the AnnotationKeyConnectorVerbose
// annotation set to `true`.
func IsVerbose(o metav1.Object) bool {
	return o.GetAnnotations()[AnnotationKeyConnectorVerbose] == "true"
}

// IsActionAllowed determines if action is allowed to be performed on Object
func IsActionAllowed(o metav1.Object, action string) bool {
	p := o.GetAnnotations()[AnnotationKeyManagementPolicy]
	if len(p) == 0 {
		p = ManagementPolicyDefault
	}

	if action == ActionCreate || action == ActionUpdate {
		return p == ManagementPolicyDefault || p == ManagementPolicyObserveCreateUpdate
	}

	// ObjectActionDelete
	return p == ManagementPolicyDefault || p == ManagementPolicyObserveDelete
}
