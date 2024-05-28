package components

import (
	"strconv"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Romeo struct {
	ns       *corev1.Namespace
	randName *random.RandomString
	pvc      *corev1.PersistentVolumeClaim
	dep      *appsv1.Deployment
	svc      *corev1.Service

	Port      pulumi.StringOutput
	Namespace pulumi.StringOutput
	ClaimName pulumi.StringOutput
}

type RomeoArgs struct{}

func NewRomeo(ctx *pulumi.Context, args *RomeoArgs, opts ...pulumi.ResourceOption) (*Romeo, error) {
	if args == nil {
		args = &RomeoArgs{}
	}

	romeo := &Romeo{}
	if err := romeo.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	romeo.outputs()

	return romeo, nil
}

func (romeo *Romeo) provision(ctx *pulumi.Context, _ *RomeoArgs, opts ...pulumi.ResourceOption) (err error) {
	// Generate unique (random enough) PVC name
	romeo.randName, err = random.NewRandomString(ctx, "romeo-name", &random.RandomStringArgs{
		Length:  pulumi.Int(8),
		Special: pulumi.Bool(false),
		Numeric: pulumi.Bool(false),
		Upper:   pulumi.Bool(false),
	}, opts...)
	if err != nil {
		return
	}

	// Provision K8s resource
	// => Create labels
	labels := pulumi.StringMap{
		"app":  pulumi.String("ctfer-io_romeo"),
		"name": romeo.randName.Result,
	}

	// => Namespace
	romeo.ns, err = corev1.NewNamespace(ctx, "romeo-ns", &corev1.NamespaceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Labels: labels,
		},
	}, opts...)
	if err != nil {
		return
	}

	// => PVC
	romeo.pvc, err = corev1.NewPersistentVolumeClaim(ctx, "romeo-pvc", &corev1.PersistentVolumeClaimArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: romeo.ns.Metadata.Name(),
			Labels:    labels,
			Name:      romeo.randName.Result,
		},
		Spec: corev1.PersistentVolumeClaimSpecArgs{
			StorageClassName: pulumi.String("longhorn"),
			AccessModes: pulumi.ToStringArray([]string{
				"ReadWriteMany",
			}),
			Resources: corev1.VolumeResourceRequirementsArgs{
				Requests: pulumi.ToStringMap(map[string]string{
					"storage": "1Gi",
				}),
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => Deployment
	romeo.dep, err = appsv1.NewDeployment(ctx, "romeo-dep", &appsv1.DeploymentArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: romeo.ns.Metadata.Name(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpecArgs{
			Selector: metav1.LabelSelectorArgs{
				MatchLabels: labels,
			},
			Replicas: pulumi.Int(1),
			Template: corev1.PodTemplateSpecArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: romeo.ns.Metadata.Name(),
					Labels:    labels,
				},
				Spec: corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Image:           pulumi.String("registry.dev1.ctfer-io.lab/ctferio/romeo:dev"),
							ImagePullPolicy: pulumi.String("Always"),
							Name:            pulumi.String("romeo"),
							Ports: corev1.ContainerPortArray{
								corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(8080),
									Name:          pulumi.String("api"),
								},
							},
							Env: corev1.EnvVarArray{
								corev1.EnvVarArgs{
									Name:  pulumi.String("COVERDIR"),
									Value: pulumi.String("/tmp/coverdir"),
								},
							},
							VolumeMounts: corev1.VolumeMountArray{
								corev1.VolumeMountArgs{
									Name:      pulumi.String("coverdir"),
									MountPath: pulumi.String("/tmp/coverdir"),
								},
							},
						},
					},
					Volumes: corev1.VolumeArray{
						corev1.VolumeArgs{
							Name: pulumi.String("coverdir"),
							PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSourceArgs{
								ClaimName: romeo.pvc.Metadata.Name().Elem(),
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

	// => Service (expose Romeo)
	romeo.svc, err = corev1.NewService(ctx, "romeo-svc", &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: romeo.ns.Metadata.Name(),
			Name:      pulumi.String("romeo-svc"),
			Labels:    labels,
		},
		Spec: &corev1.ServiceSpecArgs{
			Type:     pulumi.String("NodePort"),
			Selector: labels,
			Ports: corev1.ServicePortArray{
				corev1.ServicePortArgs{
					TargetPort: pulumi.Int(8080),
					Port:       pulumi.Int(8080),
					Name:       pulumi.String("api"),
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	return
}

func (romeo *Romeo) outputs() {
	romeo.ClaimName = romeo.randName.Result
	romeo.Namespace = romeo.ns.Metadata.Name().Elem()
	romeo.Port = romeo.svc.Spec.ApplyT(func(spec corev1.ServiceSpec) string {
		if len(spec.Ports) == 0 || spec.Ports[0].NodePort == nil {
			return ""
		}
		return strconv.Itoa(*spec.Ports[0].NodePort)
	}).(pulumi.StringOutput)
}
