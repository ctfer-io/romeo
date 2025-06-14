package parts_test

import (
	"testing"

	"github.com/ctfer-io/romeo/environment/deploy/parts"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mocks struct{}

func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	return args.Name + "_id", args.Inputs, nil
}

func (mocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	return args.Args, nil
}

func Test_U_RomeoEnvironment(t *testing.T) {
	t.Parallel()

	var tests = map[string]struct {
		Args      *parts.RomeoEnvironmentArgs
		ExpectErr bool
	}{
		"nil": {
			Args: nil,
		},
		"empty-args": {
			Args: &parts.RomeoEnvironmentArgs{},
		},
		"local": {
			Args: &parts.RomeoEnvironmentArgs{
				Registry: pulumi.String("localhost:5000"),
			},
		},
	}

	for testname, tt := range tests {
		t.Run(testname, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				_, err := parts.NewRomeoEnvironment(ctx, "romeo-test", tt.Args)
				if tt.ExpectErr {
					require.Error(err)
				} else {
					require.NoError(err)
				}

				return nil
			}, pulumi.WithMocks("project", "stack", mocks{}))
			assert.NoError(err)
		})
	}
}
