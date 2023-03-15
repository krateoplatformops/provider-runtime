package helpers

import (
	"os"
	"strings"
)

const (
	envKubernetesServicePort = "KUBERNETES_SERVICE_PORT"
)

// tcp/10.0.7.193:80
func FixKubernetesServicePort() {
	ksp := os.Getenv(envKubernetesServicePort)
	if !strings.HasPrefix(ksp, "tcp") {
		return
	}

	// hack to fix wrong kubernetes service port env var
	idx := strings.LastIndex(ksp, ":")
	if idx != -1 {
		os.Setenv(envKubernetesServicePort, ksp[idx+1:])
	}
}
