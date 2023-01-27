package resource

import (
	"context"

	prv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errExtractEnv           = "cannot extract from environment variable when none specified"
	errExtractSecretKey     = "cannot extract from secret key when none specified"
	errGetCredentialsSecret = "cannot get credentials secret"
)

// EnvLookupFn looks up an environment variable.
type EnvLookupFn func(string) string

// ExtractEnv extracts credentials from an environment variable.
func ExtractEnv(ctx context.Context, e EnvLookupFn, s prv1.CredentialSelectors) ([]byte, error) {
	if s.Env == nil {
		return nil, errors.New(errExtractEnv)
	}
	return []byte(e(s.Env.Name)), nil
}

// ExtractSecret extracts credentials from a Kubernetes secret.
func ExtractSecret(ctx context.Context, client client.Client, s prv1.CredentialSelectors) ([]byte, error) {
	if s.SecretRef == nil {
		return nil, errors.New(errExtractSecretKey)
	}
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: s.SecretRef.Namespace, Name: s.SecretRef.Name}, secret); err != nil {
		return nil, errors.Wrap(err, errGetCredentialsSecret)
	}
	return secret.Data[s.SecretRef.Key], nil
}
