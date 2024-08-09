package resource

import (
	"context"

	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// A ManagedKind contains the type metadata for a kind of managed resource.
type ManagedKind schema.GroupVersionKind

// MustCreateObject returns a new Object of the supplied kind. It panics if the
// kind is unknown to the supplied ObjectCreator.
func MustCreateObject(kind schema.GroupVersionKind, oc runtime.ObjectCreater) runtime.Object {
	obj, err := oc.New(kind)
	if err != nil {
		panic(err)
	}
	return obj
}

// An ErrorIs function returns true if an error satisfies a particular condition.
type ErrorIs func(err error) bool

// Ignore any errors that satisfy the supplied ErrorIs function by returning
// nil. Errors that do not satisfy the supplied function are returned unmodified.
func Ignore(is ErrorIs, err error) error {
	if is(err) {
		return nil
	}
	return err
}

// IgnoreAny ignores errors that satisfy any of the supplied ErrorIs functions
// by returning nil. Errors that do not satisfy any of the supplied functions
// are returned unmodified.
func IgnoreAny(err error, is ...ErrorIs) error {
	for _, f := range is {
		if f(err) {
			return nil
		}
	}
	return err
}

// IgnoreNotFound returns the supplied error, or nil if the error indicates a
// Kubernetes resource was not found.
func IgnoreNotFound(err error) error {
	return Ignore(kerrors.IsNotFound, err)
}

// IsAPIError returns true if the given error's type is of Kubernetes API error.
func IsAPIError(err error) bool {
	_, ok := err.(kerrors.APIStatus) //nolint: errorlint // we assert against the kerrors.APIStatus Interface which does not implement the error interface
	return ok
}

// IsAPIErrorWrapped returns true if err is a K8s API error, or recursively wraps a K8s API error
func IsAPIErrorWrapped(err error) bool {
	return IsAPIError(errors.Cause(err))
}

func GetConfigMapValue(ctx context.Context, kube client.Client, ref *commonv1.ConfigMapKeySelector) (string, error) {
	if ref == nil {
		return "", errors.New("no configmap referenced")
	}

	cm := &corev1.ConfigMap{}
	err := kube.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, cm)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get %s configmap", ref.Name)
	}

	return string(cm.Data[ref.Key]), nil
}

func GetSecret(ctx context.Context, k client.Client, ref *commonv1.SecretKeySelector) (string, error) {
	if ref == nil {
		return "", errors.New("no credentials secret referenced")
	}

	s := &corev1.Secret{}
	if err := k.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, s); err != nil {
		return "", errors.Wrapf(err, "cannot get %s secret", ref.Name)
	}

	return string(s.Data[ref.Key]), nil
}
