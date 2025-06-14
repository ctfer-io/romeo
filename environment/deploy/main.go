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
			Kubeconfig: cfg.Kubeconfig,
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

		// Deploy a Romeo instance
		romeo, err := parts.NewRomeoEnvironment(ctx, "deploy", &parts.RomeoEnvironmentArgs{
			Namespace:        pulumi.String(cfg.Namespace),
			Tag:              pulumi.String(cfg.Tag),
			StorageClassName: pulumi.String(cfg.StorageClassName),
			StorageSize:      pulumi.String(cfg.StorageSize),
			ClaimName: func() (s pulumi.StringInput) {
				if cfg.ClaimName != "" {
					s = pulumi.String(cfg.ClaimName)
				}
				return
			}(),
			PVCAccessModes: pulumi.ToStringArray([]string{
				cfg.PVCAccessMode,
			}),
			Registry: pulumi.String(cfg.Registry),
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
	Kubeconfig       pulumi.StringOutput
	Namespace        string
	Tag              string
	StorageClassName string
	StorageSize      string
	ClaimName        string
	PVCAccessMode    string
	Registry         string
}

func loadConfig(ctx *pulumi.Context) *Config {
	cfg := config.New(ctx, "env")
	return &Config{
		Kubeconfig:       cfg.GetSecret("kubeconfig"),
		Namespace:        cfg.Get("namespace"),
		Tag:              cfg.Get("tag"),
		StorageClassName: cfg.Get("storage-class-name"),
		StorageSize:      cfg.Get("storage-size"),
		ClaimName:        cfg.Get("claim-name"),
		PVCAccessMode:    cfg.Get("pvc-access-mode"),
		Registry:         cfg.Get("registry"),
	}
}
