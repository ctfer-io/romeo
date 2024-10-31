package parts

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"text/template"

	"github.com/pkg/errors"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	// RomeoInstall contains the RBAC ressources required by Romeo environments.
	// In summary, it contains a Role, a ServiceAccount, a RoleBinding and a Secret
	// to create a kubeconfig. This last should be passed to the workflow that uses
	// Romeo, enabling it to deploy ephemeral resources on demand.
	RomeoInstall struct {
		pulumi.ResourceState

		ns   *corev1.Namespace
		role *rbacv1.Role
		sa   *corev1.ServiceAccount
		rb   *rbacv1.RoleBinding
		sec  *corev1.Secret

		// Kubeconfig to store in the workflow secrets. Pass this to the Romeo
		// steps for deploying ephemeral environments.
		Kubeconfig pulumi.StringOutput

		// Namespace in which the resources have been created.
		// By nature, should be considered ephemeral thus should not contain
		// any production charge (only for tests/IVV purposes).
		Namespace pulumi.StringOutput
	}

	// RomeoInstallArgs contains all the arguments to setup Romeo environments.
	RomeoInstallArgs struct {
		// Namespace in which to sets up the Romeo environments.
		Namespace pulumi.StringPtrInput
		namespace pulumi.StringOutput

		// ApiServer URL to reach the Kubernetes cluster at.
		// Will be used to create the kubeconfig (output).
		ApiServer pulumi.StringInput
	}
)

// NewRomeoInstall deploys resources on on Kubernetes for Romeo environments.
// The RomeoInstall variable could be reused as a Pulumi ressource i.e. could
// be a dependency, consumes inputs and produces outputs, etc.
func NewRomeoInstall(ctx *pulumi.Context, name string, args *RomeoInstallArgs, opts ...pulumi.ResourceOption) (*RomeoInstall, error) {
	if args == nil {
		return nil, errors.New("romeo install does not support default arguments")
	}
	if args.ApiServer == pulumi.String("") {
		return nil, errors.New("api-server is required")
	}

	renv := &RomeoInstall{}
	if err := ctx.RegisterComponentResource("ctfer-io:romeo:install", name, renv, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(renv))
	if err := renv.provision(ctx, args, opts...); err != nil {
		return nil, errors.Wrap(err, "provisioning Romeo install")
	}
	renv.outputs(args)

	return renv, nil
}

func (renv *RomeoInstall) provision(ctx *pulumi.Context, args *RomeoInstallArgs, opts ...pulumi.ResourceOption) (err error) {
	// Deploy Kubernetes resources

	// => Namespace (deploy one if none specified)
	if args.Namespace == nil || args.Namespace == pulumi.String("") {
		renv.ns, err = corev1.NewNamespace(ctx, "romeo-ns", &corev1.NamespaceArgs{
			Metadata: metav1.ObjectMetaArgs{
				Labels: pulumi.StringMap{
					"app.kubernetes.io/component": pulumi.String("install"),
					"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				},
			},
		}, opts...)
		if err != nil {
			return
		}

		args.namespace = renv.ns.Metadata.Namespace().Elem()
	} else {
		args.namespace = args.Namespace.ToStringPtrOutput().Elem()
	}

	// => Role
	renv.role, err = rbacv1.NewRole(ctx, "romeo-role", &rbacv1.RoleArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		// TODO  +  kubernetes:core/v1:PersistentVolumeClaim romeo-pvc creating (0s) warning: getting storagclass "longhorn": storageclasses.storage.k8s.io "longhorn" is forbidden: User "system:serviceaccount:romeo:romeo-sa-fa059d0f" cannot get resource "storageclasses" in API group "storage.k8s.io" at the cluster scope
		Rules: rbacv1.PolicyRuleArray{
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.ToStringArray([]string{
					"",
				}),
				Verbs: pulumi.ToStringArray([]string{
					"create",
					"delete",
					"get",
					"patch",
					"list",
					"watch",
				}),
				Resources: pulumi.ToStringArray([]string{
					"persistentvolumeclaims",
					"services",
					"endpoints",
					"events",
					"pods",
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
					"patch",
					"list",
					"watch",
				}),
				Resources: pulumi.ToStringArray([]string{
					"deployments",
					"replicasets",
				}),
			},
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.ToStringArray([]string{
					"storage.k8s.io",
				}),
				Verbs: pulumi.ToStringArray([]string{
					"get",
				}),
				Resources: pulumi.ToStringArray([]string{
					"storageclasses",
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
			Namespace: args.namespace,
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
			Namespace: args.namespace,
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
				Namespace: args.namespace,
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => Secret
	renv.sec, err = corev1.NewSecret(ctx, "sa-secret", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.namespace,
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

func (renv *RomeoInstall) outputs(args *RomeoInstallArgs) {
	renv.Namespace = args.namespace
	renv.Kubeconfig = pulumi.All(renv.sec.Data, args.ApiServer, args.namespace).ApplyT(func(all []any) string {
		data := all[0].(map[string]string)
		apiServer := all[1].(string)
		ns := all[2].(string)

		token, _ := base64.StdEncoding.DecodeString(data["token"])

		values := &KubeconfigTemplateValues{
			CaCrt:     data["ca.crt"],
			ApiServer: apiServer,
			Namespace: ns,
			Token:     string(token),
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
	ApiServer string
	Namespace string
	Token     string
}
