package e2e

import "os"

var (
	runE2E = os.Getenv("RUN_E2E") != ""

	// kubectlPath is the path of kubectl. e2e/Makefile specifies it with the KUBECTL environment variable.
	kubectlPath string
)

func init() {
	kubectlPath = os.Getenv("KUBECTL")
	if kubectlPath == "" {
		kubectlPath = "kubectl"
	}
}
