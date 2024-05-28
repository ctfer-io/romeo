package iac

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/urfave/cli/v2"
)

func Up(ctx *cli.Context) error {
	fmt.Println("Getting Pulumi stack")
	stack, err := getStack(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("Spinning up Romeo resources")
	res, err := stack.Up(ctx.Context, optup.ProgressStreams(os.Stdout))
	if err != nil {
		return err
	}

	fmt.Println("Exporting Pulumi state and outputs")

	// Export Pulumi state
	if err := exportState(ctx.Context, stack); err != nil {
		return err
	}

	// Export step outputs
	port := res.Outputs["port"].Value.(string)
	namespace := res.Outputs["namespace"].Value.(string)
	claimName := res.Outputs["claimName"].Value.(string)

	f, err := os.Open(os.Getenv("GITHUB_OUTPUT"))
	if err != nil {
		return errors.Wrap(err, "github outputs file")
	}
	defer f.Close()

	_, _ = fmt.Fprintf(f, "port=%s\n", port)
	_, _ = fmt.Fprintf(f, "namespace=%s", namespace)
	_, _ = fmt.Fprintf(f, "claimName=%s", claimName)

	return nil
}
