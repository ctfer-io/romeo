package iac

import (
	"context"
	"os"

	"github.com/ctfer-io/romeo/deploy/components"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	StateFile = "/github/workspace/romeo.json"
)

func getStack(ctx context.Context) (auto.Stack, error) {
	ws, err := auto.NewLocalWorkspace(ctx,
		auto.Program(entrypoint),
		auto.Project(workspace.Project{
			Name:    "romeo-iac",
			Runtime: workspace.NewProjectRuntimeInfo("go", map[string]any{}),
		}),
		auto.EnvVars(map[string]string{
			"PULUMI_CONFIG_PASSPHRASE": "",
		}),
	)
	if err != nil {
		return auto.Stack{}, errors.Wrap(err, "creating local workspace")
	}

	stackName := auto.FullyQualifiedStackName("organization", "romeo-iac", "gha")
	stack, err := auto.UpsertStack(ctx, stackName, ws)
	if err != nil {
		return auto.Stack{}, errors.Wrap(err, "creating stack")
	}

	return stack, nil
}

func exportState(ctx context.Context, stack auto.Stack) error {
	udp, err := stack.Export(ctx)
	if err != nil {
		return errors.Wrap(err, "export stack")
	}
	if err := os.WriteFile(StateFile, udp.Deployment, 0644); err != nil {
		return errors.Wrap(err, "writing state file")
	}
	return nil
}

func importState(ctx context.Context, stack auto.Stack) error {
	b, err := os.ReadFile(StateFile)
	if err != nil {
		return errors.Wrap(err, "reading state file")
	}
	if err := stack.Import(ctx, apitype.UntypedDeployment{
		Version:    3,
		Deployment: b,
	}); err != nil {
		return errors.Wrap(err, "import stack")
	}
	return nil
}

func entrypoint(ctx *pulumi.Context) error {
	romeo, err := components.NewRomeo(ctx, &components.RomeoArgs{})
	if err != nil {
		return errors.Wrap(err, "spinning up romeo instance")
	}

	ctx.Export("port", romeo.Port)
	ctx.Export("namespace", romeo.Namespace)
	ctx.Export("claimName", romeo.ClaimName)

	return nil
}
