package main

import (
	"github.com/ctfer-io/romeo/environment/deploy/parts"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadConfig(ctx)

		// Build Kubernetes provider
		pv, err := kubernetes.NewProvider(ctx, "provider", &kubernetes.ProviderArgs{
			Kubeconfig: pulumi.String(cfg.Kubeconfig),
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
		}

		// Deploy a Romeo instance
		romeo, err := parts.NewRomeoEnvironment(ctx, "deploy", &parts.RomeoEnvironmentArgs{
			Namespace:        pulumi.String(cfg.Namespace),
			Tag:              pulumi.String(cfg.Tag),
			StorageClassName: pulumi.String(cfg.StorageClassName),
			StorageSize:      pulumi.String(cfg.StorageSize),
			ClaimName:        pulumi.String(cfg.ClaimName),
			PVCAccessModes: pulumi.ToStringArray([]string{
				cfg.PVCAccessMode,
			}),
			PrivateRegistry: pulumi.String(cfg.PrivateRegistry),
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

type Config struct {
	Kubeconfig       string
	Namespace        string
	Tag              string
	StorageClassName string
	StorageSize      string
	ClaimName        string
	PVCAccessMode    string
	PrivateRegistry  string
}

func loadConfig(ctx *pulumi.Context) *Config {
	cfg := config.New(ctx, "")
	return &Config{
		Kubeconfig:       cfg.Require("kubeconfig"),
		Namespace:        cfg.Get("namespace"),
		Tag:              cfg.Get("tag"),
		StorageClassName: cfg.Get("storage-class-name"),
		StorageSize:      cfg.Get("storage-size"),
		ClaimName:        cfg.Get("claim-name"),
		PVCAccessMode:    cfg.Get("pvc-access-mode"),
		PrivateRegistry:  cfg.Get("private-registry"),
	}
}
