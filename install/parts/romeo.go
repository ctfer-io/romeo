package parts

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/pkg/errors"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	// RomeoEnvironment contains the RBAC ressources required by Romeo deployments.
	// In summary, it contains a Role, a ServiceAccount, a RoleBinding and a Secret
	// to create a kubeconfig. This last should be passed to the workflow that uses
	// Romeo, enabling it to deploy ephemeral resources on demand.
	RomeoEnvironment struct {
		pulumi.ResourceState

		role *rbacv1.Role
		sa   *corev1.ServiceAccount
		rb   *rbacv1.RoleBinding
		sec  *corev1.Secret

		// Kubeconfig to store in the workflow secrets. Pass this to the Romeo
		// steps for deploying ephemeral environments.
		Kubeconfig pulumi.StringOutput
	}

	// RomeoEnvironmentArgs contains all the arguments to setup a Romeo environment.
	RomeoEnvironmentArgs struct {
		// Namespace in which to sets up the Romeo environment.
		Namespace pulumi.String
		// Server URL to reach the Kubernetes cluster at.
		// Will be used to create the kubeconfig (output).
		Server string
	}
)

// NewRomeoEnvironment deploys a Romeo environment on Kubernetes.
// The RomeoEnvironment variable could be reused as a Pulumi ressource i.e. could
// be a dependency, consumes inputs and produces outputs, etc.
func NewRomeoEnvironment(ctx *pulumi.Context, name string, args *RomeoEnvironmentArgs, opts ...pulumi.ResourceOption) (*RomeoEnvironment, error) {
	if args == nil {
		return nil, errors.New("romeo environment does not support default arguments")
	}
	if args.Namespace == pulumi.String("") {
		return nil, errors.New("namespace is required")
	}
	if args.Server == "" {
		return nil, errors.New("server is required")
	}

	renv := &RomeoEnvironment{}
	if err := ctx.RegisterComponentResource("ctfer-io:romeo:environment", name, renv, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(renv))
	if err := renv.provision(ctx, args, opts...); err != nil {
		return nil, errors.Wrap(err, "provisioning Romeo environment")
	}
	renv.outputs(args)

	return renv, nil
}

func (renv *RomeoEnvironment) provision(ctx *pulumi.Context, args *RomeoEnvironmentArgs, opts ...pulumi.ResourceOption) (err error) {
	// Deploy Kubernetes resources

	// => Role
	renv.role, err = rbacv1.NewRole(ctx, "romeo-role", &rbacv1.RoleArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		Rules: rbacv1.PolicyRuleArray{
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.ToStringArray([]string{
					"",
				}),
				Verbs: pulumi.ToStringArray([]string{
					"create",
					"delete",
					"get",
					// "list",
					// "watch",
				}),
				Resources: pulumi.ToStringArray([]string{
					"persistentvolumeclaims",
				}),
			},
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.ToStringArray([]string{
					"apps",
				}),
				Verbs: pulumi.ToStringArray([]string{
					"create",
					"delete",
					"get",
					// "list",
					// "watch",
				}),
				Resources: pulumi.ToStringArray([]string{
					"deployments",
					"services",
				}),
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => ServiceAccount
	renv.sa, err = corev1.NewServiceAccount(ctx, "romeo-sa", &corev1.ServiceAccountArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => RoleBinding
	renv.rb, err = rbacv1.NewRoleBinding(ctx, "romeo-role-binding", &rbacv1.RoleBindingArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("romeo"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		RoleRef: rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("Role"),
			Name:     renv.role.Metadata.Name().Elem(),
		},
		Subjects: rbacv1.SubjectArray{
			rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      renv.sa.Metadata.Name().Elem(),
				Namespace: args.Namespace,
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => Secret
	renv.sec, err = corev1.NewSecret(ctx, "sa-secret", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Annotations: pulumi.StringMap{
				"kubernetes.io/service-account.name": renv.sa.Metadata.Name().Elem(),
			},
		},
		Type: pulumi.String("kubernetes.io/service-account-token"),
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (renv *RomeoEnvironment) outputs(args *RomeoEnvironmentArgs) {
	renv.Kubeconfig = renv.sec.Data.ApplyT(func(data map[string]string) string {
		values := &KubeconfigTemplateValues{
			CaCrt:     data["ca.crt"],
			Server:    args.Server,
			Namespace: data["namespace"],
			Token:     data["token"],
		}
		buf := &bytes.Buffer{}
		if err := kubeTemplate.Execute(buf, values); err != nil {
			panic(err)
		}
		return buf.String()
	}).(pulumi.StringOutput)
}

var (
	//go:embed kubeconfig-template.yaml
	rawKubeTemplate string
	kubeTemplate    *template.Template
)

func init() {
	t, err := template.New("kubeconfig").Parse(rawKubeTemplate)
	if err != nil {
		panic(err)
	}
	kubeTemplate = t
}

// KubeconfigTemplateValues contains the inputs to build a
type KubeconfigTemplateValues struct {
	CaCrt     string
	Server    string
	Namespace string
	Token     string
}
