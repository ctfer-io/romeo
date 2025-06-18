package integration

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"testing"

	apiv1 "github.com/ctfer-io/romeo/webserver/api/v1"
	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_I_Coverout(t *testing.T) {
	cwd, _ := os.Getwd()
	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Quick:       true,
		SkipRefresh: true,
		Dir:         path.Join(cwd, ".."),
		Config: map[string]string{
			"namespace":       os.Getenv("NAMESPACE"),
			"tag":             os.Getenv("TAG"),
			"registry":        os.Getenv("REGISTRY"),
			"claim-name":      os.Getenv("CLAIM_NAME"),
			"pvc-access-mode": "ReadWriteOnce", // don't need to scale (+ not possible with kind in CI)
			"expose":          "true",          // make API externally reachable
			"harden":          "true",          // we test Romeo in a hardened env -> need the netpol
		},
		Secrets: map[string]string{
			"kubeconfig": os.Getenv("KUBECONFIG"),
		},
		ExtraRuntimeValidation: func(t *testing.T, stack integration.RuntimeValidationStackInfo) {
			// Issue API call
			server := fmt.Sprintf("%s:%0.f", os.Getenv("SERVER"), stack.Outputs["port"])
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/api/v1/coverout", server), nil)
			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = res.Body.Close()
			}()

			// Unmarshal response
			var r apiv1.CoveroutResponse
			err = yaml.NewDecoder(res.Body).Decode(&r)
			require.NoError(t, err)

			// Extract coverage infos
			dpath, _ := os.MkdirTemp("", "*")
			err = apiv1.Decode(r.Merged, dpath)
			require.NoError(t, err)

			// No need to check for coverage validity, it will  naturally be
			// checked by later steps (in CI).
		},
	})
}
