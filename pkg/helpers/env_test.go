package helpers

import (
	"os"
	"testing"
)

func TestFixKubernetesServicePort(t *testing.T) {
	testcases := []struct {
		input string
		want  string
	}{
		{"443", "443"},
		{"tcp/10.0.7.193:80", "80"},
		{"tcp/10.0.7.193:443", "443"},
	}

	for _, tc := range testcases {
		os.Setenv(envKubernetesServicePort, tc.input)
		FixKubernetesServicePort()
		got := os.Getenv(envKubernetesServicePort)
		if got != tc.want {
			t.Fatalf("got: %s, want: %s", got, tc.want)
		}
	}
}
