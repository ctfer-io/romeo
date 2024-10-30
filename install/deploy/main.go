package main

import (
	"github.com/ctfer-io/romeo/install/deploy/parts"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "romeo-install")

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

		// Install Romeo
		renv, err := parts.NewRomeoInstall(ctx, "install", &parts.RomeoInstallArgs{
			Namespace: pulumi.String(cfg.Get("namespace")),
			ApiServer: pulumi.String(cfg.Get("api-server")),
		}, opts...)
		if err != nil {
			return err
		}

		// Export romeo outputs
		ctx.Export("kubeconfig", renv.Kubeconfig)

		return nil
	})
}
