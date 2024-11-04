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
		cfg := config.New(ctx, "romeo-install")

		// Build Kubernetes provider
		kubeconfig := cfg.Require("kubeconfig")
		pv, err := kubernetes.NewProvider(ctx, "provider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(kubeconfig),
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
		}

		// Get api-server
		apiServer := cfg.Get("api-server")
		if apiServer == "" {
			apiServer, err = extractApiServer(kubeconfig)
			if err != nil {
				return err
			}
		}

		// Install Romeo
		rist, err := parts.NewRomeoInstall(ctx, "install", &parts.RomeoInstallArgs{
			Namespace: pulumi.String(cfg.Get("namespace")),
			ApiServer: pulumi.String(apiServer),
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

func extractApiServer(kubeconfig string) (string, error) {
	kc := &PartialKubeconfig{}
	if err := yaml.Unmarshal([]byte(kubeconfig), kc); err != nil {
		return "", err
	}
	if len(kc.Clusters) != 1 {
		return "", errors.New("could not infer api-server from kubeconfig")
	}
	return kc.Clusters[0].Cluster.Server, nil
}
