package main

import (
	"github.com/ctfer-io/romeo/install/deploy/parts"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "romeo-install")

		renv, err := parts.NewRomeoInstall(ctx, "install", &parts.RomeoInstallArgs{
			Namespace: pulumi.String(cfg.Require("namespace")),
			ApiServer: cfg.Require("api-server"),
		})
		if err != nil {
			return err
		}

		ctx.Export("kubeconfig", renv.Kubeconfig)

		return nil
	})
}
