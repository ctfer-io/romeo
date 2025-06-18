package parts

import (
	"fmt"

	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	// Namespace is an isolated and secured Kubernetes namespace, with security
	// annotations for enforce and warn to baseline, and versions to latest.
	// It is deployed with a basic set of network policies that ensure the network
	// isolation toward adjacent namespaces, and deny all non-explicitly-granted
	// traffic.
	Namespace struct {
		pulumi.ResourceState

		rd *random.RandomString
		ns *corev1.Namespace

		// Name of the namespace. Is going to be appended a 8-char random string
		// for parallel deployments within a single Kubernetes cluster (e.g. CI).
		// Pass it to the namespacable resources to deploy them into.
		Name pulumi.StringOutput

		// Labels of the namespace.
		Labels pulumi.StringMapOutput
	}

	NamespaceArgs struct {
		// Name is an optional value that defines the namespace name.
		Name pulumi.StringInput

		// AdditionalLabels to pass to the namespace, mostly for filtering purposes.
		AdditionalLabels pulumi.StringMapInput
	}
)

// NewNamespace creates a new [*Namespace].
func NewNamespace(ctx *pulumi.Context, name string, args *NamespaceArgs, opts ...pulumi.ResourceOption) (*Namespace, error) {
	ns := &Namespace{}

	args = ns.defaults(args)
	if err := ctx.RegisterComponentResource("ctfer-io:romeo-install:namespace", name, ns, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(ns))
	if err := ns.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	if err := ns.outputs(ctx); err != nil {
		return nil, err
	}
	return ns, nil
}

func (ns *Namespace) defaults(args *NamespaceArgs) *NamespaceArgs {
	if args == nil {
		args = &NamespaceArgs{}
	}

	if args.Name == nil {
		args.Name = pulumi.String("").ToStringOutput()
	}

	if args.AdditionalLabels == nil {
		args.AdditionalLabels = pulumi.StringMap{}.ToStringMapOutput()
	}

	return args
}

func (ns *Namespace) provision(ctx *pulumi.Context, args *NamespaceArgs, opts ...pulumi.ResourceOption) (err error) {
	if args.Name != nil {
		ns.rd, err = random.NewRandomString(ctx, "ns-suffix", &random.RandomStringArgs{
			Length:  pulumi.Int(8),
			Lower:   pulumi.Bool(true),
			Numeric: pulumi.Bool(false),
			Special: pulumi.Bool(false),
			Upper:   pulumi.Bool(false),
		}, opts...)
		if err != nil {
			return
		}
	}

	ns.ns, err = corev1.NewNamespace(ctx, "ns", &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Name: pulumi.All(args.Name, ns.rd.Result).ApplyT(func(all []any) string {
				name, ok := all[0].(string)
				if !ok || name == "" {
					return "" // will be defaulted by Kubernetes
				}
				return fmt.Sprintf("%s-%s", name, all[1])
			}).(pulumi.StringOutput),
			Labels: args.AdditionalLabels.ToStringMapOutput().ApplyT(func(labels map[string]string) map[string]string {
				// Use the additional labels as a base, add/overwrite our own labels
				labels["pod-security.kubernetes.io/enforce"] = "baseline"
				labels["pod-security.kubernetes.io/enforce-version"] = "latest"
				labels["pod-security.kubernetes.io/warn"] = "baseline"
				labels["pod-security.kubernetes.io/warn-version"] = "latest"
				return labels
			}).(pulumi.StringMapOutput),
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (ns *Namespace) outputs(ctx *pulumi.Context) error {
	ns.Name = ns.ns.Metadata.Name().Elem()
	ns.Labels = ns.ns.Metadata.Labels()

	return ctx.RegisterResourceOutputs(ns, pulumi.Map{
		"name":   ns.Name,
		"labels": ns.Labels,
	})
}
