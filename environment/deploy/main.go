package main

import (
	"github.com/ctfer-io/romeo/environment/deploy/parts"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "romeo-environment")

		// Build Kubernetes provider
		pv, err := kubernetes.NewProvider(ctx, "provider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(cfg.Require("kubeconfig")),
			// No need to configure the namespace, will be enforced by the kubeconfig.
			// If not, will fall into "default".
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
		}

		// Deploy a Romeo instance
		romeo, err := parts.NewRomeoEnvironment(ctx, "deploy", &parts.RomeoEnvironmentArgs{
			Tag:       pulumi.String(cfg.Get("tag")),
			ClaimName: pulumi.StringPtrFromPtr(strPtr(cfg, "claim-name")),
		}, opts...)
		if err != nil {
			return err
		}

		// Export Romeo outputs
		ctx.Export("port", romeo.Port)
		ctx.Export("claim-name", romeo.ClaimName)

		return nil
	})
}

func strPtr(cfg *config.Config, key string) *string {
	v := cfg.Get(key)
	if v == "" {
		return nil
	}
	return &v
}
