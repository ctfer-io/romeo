package parts

import (
	"github.com/pkg/errors"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netwv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	Hardening struct {
		pulumi.ResourceState

		npol        *netwv1.NetworkPolicy
		dnspol      *netwv1.NetworkPolicy
		internspol  *netwv1.NetworkPolicy
		internetpol *netwv1.NetworkPolicy
	}

	HardeningArgs struct {
		// Name of the namespace to harden.
		Name pulumi.StringInput

		// AdditionalLabels to pass to the netpols.
		AdditionalLabels pulumi.StringMapInput
	}
)

func NewHardening(
	ctx *pulumi.Context,
	name string,
	args *HardeningArgs,
	opts ...pulumi.ResourceOption,
) (*Hardening, error) {
	if args == nil {
		return nil, errors.New("no arguments")
	}
	if args.Name == nil {
		return nil, errors.New("no namespace defined")
	}

	h := &Hardening{}
	if err := ctx.RegisterComponentResource("ctfer-io:romeo-install:hardening", name, h, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(h))
	if err := h.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Hardening) provision(ctx *pulumi.Context, args *HardeningArgs, opts ...pulumi.ResourceOption) (err error) {
	// Deny all traffic by default
	h.npol, err = netwv1.NewNetworkPolicy(ctx, "deny-all", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Name,
			Labels:    args.AdditionalLabels,
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PodSelector: metav1.LabelSelectorArgs{},
			PolicyTypes: pulumi.ToStringArray([]string{
				"Ingress",
				"Egress",
			}),
		},
	}, opts...)
	if err != nil {
		return
	}

	// Grant DNS resolution
	h.dnspol, err = netwv1.NewNetworkPolicy(ctx, "dns", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Name,
			Labels:    args.AdditionalLabels,
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PolicyTypes: pulumi.ToStringArray([]string{
				"Egress",
			}),
			PodSelector: metav1.LabelSelectorArgs{},
			Egress: netwv1.NetworkPolicyEgressRuleArray{
				netwv1.NetworkPolicyEgressRuleArgs{
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"kubernetes.io/metadata.name": pulumi.String("kube-system"),
								},
							},
							PodSelector: metav1.LabelSelectorArgs{
								MatchLabels: pulumi.StringMap{
									"k8s-app": pulumi.String("kube-dns"),
								},
							},
						},
					},
					Ports: netwv1.NetworkPolicyPortArray{
						netwv1.NetworkPolicyPortArgs{
							Port:     pulumi.Int(53),
							Protocol: pulumi.String("UDP"),
						},
						netwv1.NetworkPolicyPortArgs{
							Port:     pulumi.Int(53),
							Protocol: pulumi.String("TCP"),
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// Whatever happens (IP ranges, DNS entries) deny all traffic to adjacent
	// namespaces -> isolation by default/in depth.
	h.internspol, err = netwv1.NewNetworkPolicy(ctx, "inter-ns", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Name,
			Labels:    args.AdditionalLabels,
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PodSelector: metav1.LabelSelectorArgs{},
			PolicyTypes: pulumi.ToStringArray([]string{
				"Egress",
			}),
			Egress: netwv1.NetworkPolicyEgressRuleArray{
				netwv1.NetworkPolicyEgressRuleArgs{
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							NamespaceSelector: metav1.LabelSelectorArgs{
								MatchExpressions: metav1.LabelSelectorRequirementArray{
									metav1.LabelSelectorRequirementArgs{
										Key:      pulumi.String("kubernetes.io/metadata.name"),
										Operator: pulumi.String("NotIn"),
										Values: pulumi.StringArray{
											args.Name,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// For dependencies resolution and the use of external services, grant
	// access to internet, i.e. all IP ranges except private ones
	// (https://en.wikipedia.org/wiki/Private_network#Private_IPv4_addresses).
	h.internetpol, err = netwv1.NewNetworkPolicy(ctx, "internet", &netwv1.NetworkPolicyArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Name,
			Labels:    args.AdditionalLabels,
		},
		Spec: netwv1.NetworkPolicySpecArgs{
			PodSelector: metav1.LabelSelectorArgs{},
			PolicyTypes: pulumi.ToStringArray([]string{
				"Egress",
			}),
			Egress: netwv1.NetworkPolicyEgressRuleArray{
				netwv1.NetworkPolicyEgressRuleArgs{
					To: netwv1.NetworkPolicyPeerArray{
						netwv1.NetworkPolicyPeerArgs{
							IpBlock: netwv1.IPBlockArgs{
								Cidr: pulumi.String("0.0.0.0/0"),
								Except: pulumi.ToStringArray([]string{
									"10.0.0.0/8",     // internal Kubernetes cluster IP range
									"172.16.0.0/12",  // common internal IP range
									"192.168.0.0/16", // common internal IP range
								}),
							},
						},
					},
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}
