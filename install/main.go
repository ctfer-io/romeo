package main

import (
	"github.com/ctfer-io/romeo/install/parts"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "environment")

		renv, err := parts.NewRomeoEnvironment(ctx, "install", &parts.RomeoEnvironmentArgs{
			Namespace: pulumi.String(cfg.Require("namespace")),
			Server:    cfg.Require("server"),
		})
		if err != nil {
			return err
		}

		ctx.Export("kubeconfig", renv.Kubeconfig)

		return nil
	})
}
