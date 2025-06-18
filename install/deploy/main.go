package main

import (
	"errors"

	"github.com/ctfer-io/romeo/install/deploy/parts"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"gopkg.in/yaml.v3"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "install")

		// Build Kubernetes provider
		kubeconfig := cfg.GetSecret("kubeconfig")
		pv, err := kubernetes.NewProvider(ctx, "provider", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
			// Define timeouts to avoid waiting in CI.
			// These should be large enough (observed ~1m on GitHub Action runners).
			pulumi.Timeouts(&pulumi.CustomTimeouts{
				Create: "2m",
				Update: "2m", // should not occur
				Delete: "2m", // should not occur
			}),
		}

		// Get api-server
		apiServer := kubeconfig.ApplyT(func(kubeconfig string) (apiServer string, err error) {
			apiServer = cfg.Get("api-server")
			if apiServer == "" {
				apiServer, err = extractAPIServer(kubeconfig)
			}
			return
		}).(pulumi.StringOutput)

		// Install Romeo
		rist, err := parts.NewRomeoInstall(ctx, "install", &parts.RomeoInstallArgs{
			Namespace: pulumi.String(cfg.Get("namespace")),
			APIServer: apiServer,
			Harden:    cfg.GetBool("harden"),
		}, opts...)
		if err != nil {
			return err
		}

		// Export romeo outputs
		ctx.Export("kubeconfig", rist.Kubeconfig)
		ctx.Export("namespace", rist.Namespace)

		return nil
	})
}

type PartialKubeconfig struct {
	Clusters []struct {
		Cluster struct {
			Server string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
}

func extractAPIServer(kubeconfig string) (string, error) {
	kc := PartialKubeconfig{}
	if err := yaml.Unmarshal([]byte(kubeconfig), &kc); err != nil {
		return "", err
	}
	if len(kc.Clusters) != 1 {
		return "", errors.New("could not infer api-server from kubeconfig")
	}
	return kc.Clusters[0].Cluster.Server, nil
}
