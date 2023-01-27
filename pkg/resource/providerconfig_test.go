package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/krateoplatformops/provider-runtime/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-runtime/pkg/errors"
)

func TestExtractEnv(t *testing.T) {
	credentials := []byte("supersecretcreds")

	type args struct {
		e     EnvLookupFn
		creds xpv1.CredentialSelectors
	}

	type want struct {
		b   []byte
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"EnvVarSuccess": {
			reason: "Successful extraction of credentials from environment variable",
			args: args{
				e: func(string) string { return string(credentials) },
				creds: xpv1.CredentialSelectors{
					Env: &xpv1.EnvSelector{
						Name: "SECRET_CREDS",
					},
				},
			},
			want: want{
				b: credentials,
			},
		},
		"EnvVarFail": {
			reason: "Failed extraction of credentials from environment variable",
			args: args{
				e: func(string) string { return string(credentials) },
			},
			want: want{
				err: errors.New(errExtractEnv),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ExtractEnv(context.TODO(), tc.args.e, tc.args.creds)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\npc.ExtractEnv(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.b, got); diff != "" {
				t.Errorf("\n%s\npc.ExtractEnv(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestExtractSecret(t *testing.T) {
	errBoom := errors.New("boom")
	credentials := []byte("supersecretcreds")

	type args struct {
		client client.Client
		creds  xpv1.CredentialSelectors
	}

	type want struct {
		b   []byte
		err error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"SecretSuccess": {
			reason: "Successful extraction of credentials from Secret",
			args: args{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(o client.Object) error {
						s, _ := o.(*corev1.Secret)
						s.Data = map[string][]byte{
							"creds": credentials,
						}
						return nil
					}),
				},
				creds: xpv1.CredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "super",
							Namespace: "secret",
						},
						Key: "creds",
					},
				},
			},
			want: want{
				b: credentials,
			},
		},
		"SecretFailureNotDefined": {
			reason: "Failed extraction of credentials from Secret when key not defined",
			args:   args{},
			want: want{
				err: errors.New(errExtractSecretKey),
			},
		},
		"SecretFailureGet": {
			reason: "Failed extraction of credentials from Secret when client fails",
			args: args{
				client: &test.MockClient{
					MockGet: test.NewMockGetFn(nil, func(client.Object) error {
						return errBoom
					}),
				},
				creds: xpv1.CredentialSelectors{
					SecretRef: &xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{
							Name:      "super",
							Namespace: "secret",
						},
						Key: "creds",
					},
				},
			},
			want: want{
				err: errors.Wrap(errBoom, errGetCredentialsSecret),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ExtractSecret(context.TODO(), tc.args.client, tc.args.creds)
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\npc.ExtractSecret(...): -want error, +got error:\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.b, got); diff != "" {
				t.Errorf("\n%s\npc.ExtractSecret(...): -want, +got:\n%s\n", tc.reason, diff)
			}
		})
	}
}
