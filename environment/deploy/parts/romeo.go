package parts

import (
	"fmt"
	"strconv"

	appsv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apps/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	// RomeoEnvironment contains the ephemeral Kubernetes infrastructure for the Go
	// binaries to export their coverage info into.
	RomeoEnvironment struct {
		pulumi.ResourceState

		randName  *random.RandomString
		pvc       *corev1.PersistentVolumeClaim
		dep       *appsv1.Deployment
		svc       *corev1.Service
		coverRand *random.RandomString

		// Namespace to where Romeo is deployed.
		// You can reuse it for further tests such that deployed Go apps target
		// this namespace for their own deployment, thus can reach the
		// PersistentVolumeClaim.
		Namespace pulumi.StringOutput

		// The port to reach the Romeo instance on.
		Port pulumi.StringOutput

		// The claim name to mount in coverage-monitored Go pods for them to
		// export their coverage data.
		ClaimName pulumi.StringOutput
	}

	// RomeoEnvironmentArgs contains all the arguments to deploy a Romeo environment.
	RomeoEnvironmentArgs struct {
		Tag pulumi.StringInput
		tag pulumi.StringOutput

		ClaimName pulumi.StringPtrInput

		StorageClassName pulumi.StringPtrInput
		storageClassName pulumi.StringOutput

		StorageSize pulumi.StringPtrInput
		storageSize pulumi.StringOutput

		PVCAccessModes pulumi.StringArrayInput
		pvcAccessModes pulumi.StringArrayOutput

		Namespace pulumi.StringInput
	}
)

const (
	coverdir = "/tmp/coverdir"
)

// NewRomeoEnvironment deploys a Romeo instance on Kubernetes.
// The Romeo variable could be reused as a Pulumi ressource i.e. could
// be a dependency, consumes inputs and produces outputs, etc.
func NewRomeoEnvironment(ctx *pulumi.Context, name string, args *RomeoEnvironmentArgs, opts ...pulumi.ResourceOption) (*RomeoEnvironment, error) {
	romeo := &RomeoEnvironment{}

	args = romeo.process(args)
	if err := ctx.RegisterComponentResource("ctfer-io:romeo:environment", name, romeo, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(romeo))
	if err := romeo.provision(ctx, args, opts...); err != nil {
		return nil, err
	}
	romeo.outputs()

	return romeo, nil
}

func (romeo *RomeoEnvironment) process(args *RomeoEnvironmentArgs) *RomeoEnvironmentArgs {
	if args == nil {
		args = &RomeoEnvironmentArgs{}
	}

	// Default tag to dev
	if args.Tag == nil || args.Tag == pulumi.String("") {
		args.Tag = pulumi.String("dev").ToStringOutput()
	}
	args.tag = args.Tag.ToStringPtrOutput().Elem()

	// Default storage class name to longhorn
	if args.StorageClassName == nil || args.StorageClassName == pulumi.String("") {
		args.StorageClassName = pulumi.StringPtr("longhorn")
	}
	args.storageClassName = args.StorageClassName.ToStringPtrOutput().Elem()

	// Default storage size to 50M
	if args.StorageSize == nil || args.StorageSize == pulumi.String("") {
		args.StorageSize = pulumi.StringPtr("50M")
	}
	args.storageSize = args.StorageSize.ToStringPtrOutput().Elem()

	// Default PVC access modes to ReadWriteOnce
	if args.PVCAccessModes == nil {
		args.pvcAccessModes = pulumi.ToStringArray([]string{
			"ReadWriteOnce",
		}).ToStringArrayOutput()
	} else {
		args.pvcAccessModes = args.PVCAccessModes.ToStringArrayOutput().ApplyT(func(slc []string) []string {
			if len(slc) == 0 {
				return []string{"ReadWriteOnce"}
			}
			return slc
		}).(pulumi.StringArrayOutput)
	}

	return args
}

func (romeo *RomeoEnvironment) provision(ctx *pulumi.Context, args *RomeoEnvironmentArgs, opts ...pulumi.ResourceOption) (err error) {
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
	// => PVC
	romeo.pvc, err = corev1.NewPersistentVolumeClaim(ctx, "romeo-pvc", &corev1.PersistentVolumeClaimArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("romeo"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
			Name: romeo.randName.Result,
		},
		Spec: corev1.PersistentVolumeClaimSpecArgs{
			StorageClassName: args.storageClassName,
			AccessModes:      args.pvcAccessModes,
			Resources: corev1.VolumeResourceRequirementsArgs{
				Requests: pulumi.StringMap{
					"storage": args.storageSize,
				},
			},
		},
	}, opts...)
	if err != nil {
		return
	}

	// => Deployment
	envs := corev1.EnvVarArray{
		corev1.EnvVarArgs{
			Name:  pulumi.String("COVERDIR"),
			Value: pulumi.String(coverdir),
		},
	}
	volumeMounts := corev1.VolumeMountArray{
		corev1.VolumeMountArgs{
			Name:      pulumi.String("coverdir"),
			MountPath: pulumi.String(coverdir),
		},
	}
	volumes := corev1.VolumeArray{
		corev1.VolumeArgs{
			Name: pulumi.String("coverdir"),
			PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: romeo.pvc.Metadata.Name().Elem(),
			},
		},
	}
	if args.ClaimName != nil {
		// If coverage is turned on, export coverages in a random directory
		// that is different from coverdir (ensure no collision).
		romeo.coverRand, err = random.NewRandomString(ctx, "cover-rand", &random.RandomStringArgs{
			Length:  pulumi.Int(16),
			Lower:   pulumi.BoolPtr(true),
			Numeric: pulumi.BoolPtr(false),
			Special: pulumi.BoolPtr(false),
			Upper:   pulumi.BoolPtr(false),
		}, opts...)
		if err != nil {
			return
		}
		path := romeo.coverRand.Result.ApplyT(func(rand string) string {
			return fmt.Sprintf("/tmp/%s", rand)
		}).(pulumi.StringOutput)

		envs = append(envs, corev1.EnvVarArgs{
			Name:  pulumi.String("GOCOVERDIR"),
			Value: path,
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMountArgs{
			Name:      pulumi.String("coverages"),
			MountPath: path,
			ReadOnly:  pulumi.BoolPtr(true),
		})
		volumes = append(volumes, corev1.VolumeArgs{
			Name: pulumi.String("coverages"),
			PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: args.ClaimName.ToStringPtrOutput().Elem(),
			},
		})
	}
	romeo.dep, err = appsv1.NewDeployment(ctx, "romeo-dep", &appsv1.DeploymentArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name":      pulumi.String("romeo"),
				"app.kubernetes.io/version":   args.tag,
				"app.kubernetes.io/component": pulumi.String("romeo"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		Spec: appsv1.DeploymentSpecArgs{
			Selector: metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app.kubernetes.io/name":      pulumi.String("romeo"),
					"app.kubernetes.io/version":   args.tag,
					"app.kubernetes.io/component": pulumi.String("romeo"),
					"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				},
			},
			Replicas: pulumi.Int(1),
			Template: corev1.PodTemplateSpecArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: args.Namespace,
					Labels: pulumi.StringMap{
						"app.kubernetes.io/name":      pulumi.String("romeo"),
						"app.kubernetes.io/version":   args.tag,
						"app.kubernetes.io/component": pulumi.String("romeo"),
						"app.kubernetes.io/part-of":   pulumi.String("romeo"),
					},
				},
				Spec: corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:            pulumi.String("romeo"),
							Image:           pulumi.Sprintf("ctferio/romeo:%s", args.tag),
							ImagePullPolicy: pulumi.String("Always"),
							Ports: corev1.ContainerPortArray{
								corev1.ContainerPortArgs{
									ContainerPort: pulumi.Int(8080),
									Name:          pulumi.String("api"),
								},
							},
							Env:          envs,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
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
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String("romeo"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("NodePort"),
			Selector: pulumi.StringMap{
				"app.kubernetes.io/name":      pulumi.String("romeo"),
				"app.kubernetes.io/version":   args.tag,
				"app.kubernetes.io/component": pulumi.String("romeo"),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
			},
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

func (romeo *RomeoEnvironment) outputs() {
	romeo.Namespace = romeo.dep.Metadata.Namespace().Elem()
	romeo.ClaimName = romeo.randName.Result
	romeo.Port = romeo.svc.Spec.ApplyT(func(spec corev1.ServiceSpec) string {
		if len(spec.Ports) == 0 || spec.Ports[0].NodePort == nil {
			return ""
		}
		return strconv.Itoa(*spec.Ports[0].NodePort)
	}).(pulumi.StringOutput)
}
