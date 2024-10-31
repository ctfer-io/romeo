package parts

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	Namespace struct {
		ns *corev1.Namespace

		Name pulumi.StringOutput
	}

	NamespaceArgs struct {
		Name pulumi.StringPtrInput
	}
)

// NewNamespace creates a namespace or binds an existing one.
func NewNamespace(ctx *pulumi.Context, args *NamespaceArgs, opts ...pulumi.ResourceOption) (*Namespace, error) {
	if args == nil {
		args = &NamespaceArgs{}
	}

	ns := &Namespace{}
	if err := ns.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	ns.outputs(args)

	return ns, nil
}

func (ns *Namespace) provision(ctx *pulumi.Context, args *NamespaceArgs, opts ...pulumi.ResourceOption) (err error) {
	if args.Name == nil || args.Name == pulumi.String("") {
		ns.ns, err = corev1.NewNamespace(ctx, "romeo-install-ns", &corev1.NamespaceArgs{
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
	}

	return
}

func (ns *Namespace) outputs(args *NamespaceArgs) {
	ns.Name = pulumi.All(args.Name, ns.ns.Metadata).ApplyT(func(all []any) string {
		// Could be a pointer of a string
		if name, ok := all[0].(*string); ok && name != nil && *name != "" {
			return *name
		}
		if name, ok := all[0].(string); ok && name != "" {
			return name
		}

		meta := all[1].(metav1.ObjectMeta)
		return *meta.Name
	}).(pulumi.StringOutput)
}
