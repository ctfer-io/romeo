package iac

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/urfave/cli/v2"
)

func Down(ctx *cli.Context) error {
	url := ctx.String("url")
	coverfile := ctx.String("coverfile")

	fmt.Println("Getting Pulumi stack")
	stack, err := getStack(ctx.Context)
	if err != nil {
		return err
	}

	fmt.Println("Importing Pulumi state")
	if err := importState(ctx.Context, stack); err != nil {
		return err
	}
	if err := os.Remove(StateFile); err != nil {
		return err
	}
	outputs, err := stack.Outputs(ctx.Context)
	if err != nil {
		return err
	}

	if coverfile != "" {
		fmt.Printf("Fetching coverages and export to %s\n", coverfile)
		url = fmt.Sprintf("%s:%s", url, outputs["port"].Value.(string))
		if err := FetchCoverfile(ctx.Context, url, coverfile); err != nil {
			return err
		}
	}

	fmt.Println("Destroying Romeo resources")
	if _, err = stack.Destroy(ctx.Context, optdestroy.ProgressStreams(os.Stdout)); err != nil {
		return err
	}

	return err
}
