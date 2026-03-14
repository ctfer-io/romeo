package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	netwv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/ctfer-io/romeo/environment/deploy/parts"
)

func main() {
	_, testi := os.LookupEnv("CTFERIO_CHALL_MANAGER_INTEGRATION_TEST")

	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := loadConfig(ctx)

		// Build Kubernetes provider
		pv, err := kubernetes.NewProvider(ctx, "provider", &kubernetes.ProviderArgs{
			Kubeconfig: cfg.Kubeconfig,
		})
		if err != nil {
			return err
		}

		opts := []pulumi.ResourceOption{
			pulumi.Provider(pv),
			// Define timeouts to avoid waiting in CI.
			// These should be large enough (observed ~1m on GitHub Action runners).
			pulumi.Timeouts(&pulumi.CustomTimeouts{
				Create: "2m",
				Update: "2m", // should not occur
				Delete: "2m", // should not occur
			}),
		}

		// Deploy a Romeo instance
		romeo, err := parts.NewRomeoEnvironment(ctx, "deploy", &parts.RomeoEnvironmentArgs{
			Namespace:        pulumi.String(cfg.Namespace),
			Harden:           cfg.Harden,
			Tag:              pulumi.String(cfg.Tag),
			StorageClassName: pulumi.String(cfg.StorageClassName),
			StorageSize:      pulumi.String(cfg.StorageSize),
			ClaimName: func() (s pulumi.StringInput) {
				if cfg.ClaimName != "" {
					s = pulumi.String(cfg.ClaimName)
				}
				return
			}(),
			PVCAccessModes: pulumi.ToStringArray([]string{
				cfg.PVCAccessMode,
			}),
			Registry: pulumi.String(cfg.Registry),
		}, opts...)
		if err != nil {
			return err
		}

		if testi {
			if _, err := netwv1.NewNetworkPolicy(ctx, "allow-inside-oci", &netwv1.NetworkPolicyArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: pulumi.String(cfg.Namespace),
					Labels: pulumi.StringMap{
						"app.kubernetes.io/component": pulumi.String("chall-manager"),
						"app.kubernetes.io/part-of":   pulumi.String("chall-manager"),
						"ctfer.io/stack-name":         pulumi.String(ctx.Stack()),
					},
				},
				Spec: netwv1.NetworkPolicySpecArgs{
					PolicyTypes: pulumi.ToStringArray([]string{
						"Egress",
					}),
					PodSelector: metav1.LabelSelectorArgs{
						MatchLabels: romeo.PodLabels,
					},
					Egress: netwv1.NetworkPolicyEgressRuleArray{
						netwv1.NetworkPolicyEgressRuleArgs{
							Ports: netwv1.NetworkPolicyPortArray{
								netwv1.NetworkPolicyPortArgs{
									Port: pulumi.Int(5000), // we serve the OCI on this port
								},
							},
							To: netwv1.NetworkPolicyPeerArray{
								netwv1.NetworkPolicyPeerArgs{
									IpBlock: netwv1.IPBlockArgs{
										Cidr: pulumi.String("172.16.0.0/12"), // The CIDR the OCI registry lays into as a mirror
									},
								},
							},
						},
					},
				},
			}, opts...); err != nil {
				return errors.Wrap(err, "allowing inside OCI traffic")
			}
		}

		// Export Romeo outputs
		ctx.Export("namespace", romeo.Namespace)
		ctx.Export("port", romeo.Port)
		ctx.Export("claim-name", romeo.ClaimName)

		return nil
	})
}

type Config struct {
	Kubeconfig       pulumi.StringOutput
	Namespace        string
	Harden           bool
	Tag              string
	StorageClassName string
	StorageSize      string
	ClaimName        string
	PVCAccessMode    string
	Registry         string
}

func loadConfig(ctx *pulumi.Context) *Config {
	cfg := config.New(ctx, "env")
	return &Config{
		Kubeconfig:       cfg.GetSecret("kubeconfig"),
		Namespace:        cfg.Get("namespace"),
		Harden:           cfg.GetBool("harden"),
		Tag:              cfg.Get("tag"),
		StorageClassName: cfg.Get("storage-class-name"),
		StorageSize:      cfg.Get("storage-size"),
		ClaimName:        cfg.Get("claim-name"),
		PVCAccessMode:    cfg.Get("pvc-access-mode"),
		Registry:         cfg.Get("registry"),
	}
}
