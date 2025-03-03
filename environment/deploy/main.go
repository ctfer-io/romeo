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
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
		}

		// Deploy a Romeo instance
		romeo, err := parts.NewRomeoEnvironment(ctx, "deploy", &parts.RomeoEnvironmentArgs{
			Namespace:        pulumi.String(cfg.Get("namespace")),
			Tag:              pulumi.String(cfg.Get("tag")),
			StorageClassName: pulumi.StringPtrFromPtr(strPtr(cfg, "storage-class-name")),
			StorageSize:      pulumi.StringPtrFromPtr(strPtr(cfg, "storage-size")),
			ClaimName:        pulumi.StringPtrFromPtr(strPtr(cfg, "claim-name")),
		}, opts...)
		if err != nil {
			return err
		}

		// Export Romeo outputs
		ctx.Export("namespace", romeo.Namespace)
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
