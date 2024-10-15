package main

import (
	"github.com/ctfer-io/romeo/deploy/parts"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "romeo")

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
		romeo, err := parts.NewRomeo(ctx, "deploy", &parts.RomeoArgs{
			Tag: pulumi.String(cfg.Get("tag")),
		}, opts...)
		if err != nil {
			return err
		}

		// Export Romeo outputs
		ctx.Export("port", romeo.Port)
		ctx.Export("claimName", romeo.ClaimName)

		return nil
	})
}
