package parts

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"sync"
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

		ns  *Namespace
		h   *Hardening
		cr  *rbacv1.ClusterRole
		r   *rbacv1.Role
		sa  *corev1.ServiceAccount
		crb *rbacv1.ClusterRoleBinding
		rb  *rbacv1.RoleBinding
		sec *corev1.Secret

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
		Namespace       pulumi.StringInput
		createNamespace bool

		// APIServer URL to reach the Kubernetes cluster at.
		// Will be used to create the kubeconfig (output).
		APIServer pulumi.StringInput

		// Harden the namespace or not.
		// Deny all traffic, deny inter-namespace communications,
		// then grant DNS resolution, and grant internet communications.
		Harden bool
	}
)

// NewRomeoInstall deploys resources on on Kubernetes for Romeo environments.
// The RomeoInstall variable could be reused as a Pulumi ressource i.e. could
// be a dependency, consumes inputs and produces outputs, etc.
func NewRomeoInstall(
	ctx *pulumi.Context,
	name string,
	args *RomeoInstallArgs,
	opts ...pulumi.ResourceOption,
) (*RomeoInstall, error) {
	rist := &RomeoInstall{}

	args = rist.defaults(args)
	if err := ctx.RegisterComponentResource("ctfer-io:romeo-install:romeo", name, rist, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(rist))
	if err := rist.provision(ctx, args, opts...); err != nil {
		return nil, errors.Wrap(err, "provisioning Romeo install")
	}
	if err := rist.outputs(ctx, args); err != nil {
		return nil, err
	}
	return rist, nil
}

func (rist *RomeoInstall) defaults(args *RomeoInstallArgs) *RomeoInstallArgs {
	if args == nil {
		args = &RomeoInstallArgs{}
	}

	if args.APIServer == nil {
		args.APIServer = pulumi.String("")
	}

	args.createNamespace = args.Namespace == nil
	if args.Namespace != nil {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		args.Namespace.ToStringOutput().ApplyT(func(ns string) error {
			args.createNamespace = ns == ""
			wg.Done()
			return nil
		})
		wg.Wait()
	}

	return args
}

func (rist *RomeoInstall) provision(
	ctx *pulumi.Context,
	args *RomeoInstallArgs,
	opts ...pulumi.ResourceOption,
) (err error) {
	// Create namespace if required
	namespace := args.Namespace
	if args.createNamespace {
		rist.ns, err = NewNamespace(ctx, "romeo-install", &NamespaceArgs{
			Name: pulumi.String("romeo-install"),
			AdditionalLabels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		}, opts...)
		if err != nil {
			return err
		}
		namespace = rist.ns.Name

		if args.Harden {
			rist.h, err = NewHardening(ctx, "install-hard", &HardeningArgs{
				Name: namespace,
				AdditionalLabels: pulumi.StringMap{
					"app.kubernetes.io/component": pulumi.String("install"),
					"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				},
			}, opts...)
			if err != nil {
				return err
			}
		}
	}

	// => ClusterRole
	rist.cr, err = rbacv1.NewClusterRole(ctx, "romeo-crole", &rbacv1.ClusterRoleArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		Rules: rbacv1.PolicyRuleArray{
			rbacv1.PolicyRuleArgs{
				ApiGroups: pulumi.ToStringArray([]string{
					"storage.k8s.io",
				}),
				Verbs: pulumi.ToStringArray([]string{
					"get",
					"list",
					"watch",
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

	// => Role
	rist.r, err = rbacv1.NewRole(ctx, "romeo-role", &rbacv1.RoleArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
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
		},
	}, opts...)
	if err != nil {
		return
	}

	// => ServiceAccount
	rist.sa, err = corev1.NewServiceAccount(ctx, "romeo-sa", &corev1.ServiceAccountArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => ClusterRoleBinding
	rist.crb, err = rbacv1.NewClusterRoleBinding(ctx, "romeo-crole-binding", &rbacv1.ClusterRoleBindingArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		RoleRef: rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("ClusterRole"),
			Name:     rist.cr.Metadata.Name().Elem(),
		},
		Subjects: rbacv1.SubjectArray{
			rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      rist.sa.Metadata.Name().Elem(),
				Namespace: namespace,
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => RoleBinding
	rist.rb, err = rbacv1.NewRoleBinding(ctx, "romeo-role-binding", &rbacv1.RoleBindingArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("install"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		RoleRef: rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("Role"),
			Name:     rist.r.Metadata.Name().Elem(),
		},
		Subjects: rbacv1.SubjectArray{
			rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      rist.sa.Metadata.Name().Elem(),
				Namespace: namespace,
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => Secret
	rist.sec, err = corev1.NewSecret(ctx, "sa-secret", &corev1.SecretArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: namespace,
			Annotations: pulumi.StringMap{
				"kubernetes.io/service-account.name": rist.sa.Metadata.Name().Elem(),
			},
		},
		Type: pulumi.String("kubernetes.io/service-account-token"),
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (rist *RomeoInstall) outputs(ctx *pulumi.Context, args *RomeoInstallArgs) error {
	if args.createNamespace {
		rist.Namespace = rist.ns.Name
	} else {
		rist.Namespace = args.Namespace.ToStringOutput()
	}

	rist.Kubeconfig = pulumi.All(rist.sec.Data, args.APIServer, rist.ns.Name).ApplyT(func(all []any) string {
		data := all[0].(map[string]string)
		apiServer := all[1].(string)
		ns := all[2].(string)

		token, _ := base64.StdEncoding.DecodeString(data["token"])

		values := &KubeconfigTemplateValues{
			CaCrt:     data["ca.crt"],
			APIServer: apiServer,
			Namespace: ns,
			Token:     string(token),
		}
		buf := &bytes.Buffer{}
		if err := kubeTemplate.Execute(buf, values); err != nil {
			panic(err)
		}
		return buf.String()
	}).(pulumi.StringOutput)

	return ctx.RegisterResourceOutputs(rist, pulumi.Map{
		"namespace":  rist.Namespace,
		"kubeconfig": rist.Kubeconfig,
	})
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
	APIServer string
	Namespace string
	Token     string
}
