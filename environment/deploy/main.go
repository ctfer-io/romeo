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
			StorageClassName: pulumi.StringPtr(cfg.StorageClassName),
			StorageSize:      pulumi.StringPtr(cfg.StorageSize),
			ClaimName:        pulumi.StringPtrFromPtr(cfg.ClaimName),
			PVCAccessModes: pulumi.ToStringArray([]string{
				cfg.PVCAccessMode,
			}),
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
	Kubeconfig       string  `json:"kubeconfig"`
	Namespace        string  `json:"namespace"`
	Tag              string  `json:"tag"`
	StorageClassName string  `json:"storage-class-name"`
	StorageSize      string  `json:"storage-size"`
	ClaimName        *string `json:"claim-name"`
	PVCAccessMode    string  `json:"pvc-access-mode"`
}

func loadConfig(ctx *pulumi.Context) *Config {
	cfg := config.New(ctx, "")
	return &Config{
		Kubeconfig:       cfg.Require("kubeconfig"),
		Namespace:        cfg.Get("namespace"),
		Tag:              cfg.Get("tag"),
		StorageClassName: cfg.Get("storage-class-name"),
		StorageSize:      cfg.Get("storage-size"),
		ClaimName:        getStrPtr(cfg, "claim-name"),
		PVCAccessMode:    cfg.Get("pvc-access-mode"),
	}
}

func getStrPtr(cfg *config.Config, key string) *string {
	v := cfg.Get(key)
	if v != "" {
		return &v
	}
	return nil
}
