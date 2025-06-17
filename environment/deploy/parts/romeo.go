package parts

import (
	"fmt"
	"strings"

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
		Port pulumi.IntOutput

		// The claim name to mount in coverage-monitored Go pods for them to
		// export their coverage data.
		ClaimName pulumi.StringOutput
	}

	// RomeoEnvironmentArgs contains all the arguments to deploy a Romeo environment.
	RomeoEnvironmentArgs struct {
		Tag pulumi.StringInput
		tag pulumi.StringOutput

		ClaimName pulumi.StringInput

		StorageClassName pulumi.StringInput
		storageClassName pulumi.StringOutput

		StorageSize pulumi.StringInput
		storageSize pulumi.StringOutput

		Namespace pulumi.StringInput

		PVCAccessModes pulumi.StringArrayInput
		pvcAccessModes pulumi.StringArrayOutput

		// Registry define from where to fetch the Chall-Manager Docker images.
		// If set empty, defaults to Docker Hub.
		// Authentication is not supported, please provide it as Kubernetes-level configuration.
		Registry pulumi.StringInput
		registry pulumi.StringOutput
	}
)

const (
	coverdir                = "/etc/coverdir"
	defaultTag              = "dev"
	defaultStorageSize      = "50M"
	defaultStorageClassName = "standard"
)

// NewRomeoEnvironment deploys a Romeo instance on Kubernetes.
// The Romeo variable could be reused as a Pulumi ressource i.e. could
// be a dependency, consumes inputs and produces outputs, etc.
func NewRomeoEnvironment(
	ctx *pulumi.Context,
	name string,
	args *RomeoEnvironmentArgs,
	opts ...pulumi.ResourceOption,
) (*RomeoEnvironment, error) {
	renv := &RomeoEnvironment{}

	args = renv.defaults(args)
	if err := ctx.RegisterComponentResource("ctfer-io:romeo:environment", name, renv, opts...); err != nil {
		return nil, err
	}
	opts = append(opts, pulumi.Parent(renv))
	if err := renv.provision(ctx, name, args, opts...); err != nil {
		return nil, err
	}
	if err := renv.outputs(ctx); err != nil {
		return nil, err
	}

	return renv, nil
}

func (renv *RomeoEnvironment) defaults(args *RomeoEnvironmentArgs) *RomeoEnvironmentArgs {
	if args == nil {
		args = &RomeoEnvironmentArgs{}
	}

	// Default tag to dev
	args.tag = pulumi.String(defaultTag).ToStringOutput()
	if args.Tag != nil {
		args.tag = args.Tag.ToStringOutput().ApplyT(func(tag string) string {
			if tag == "" {
				return defaultTag
			}
			return tag
		}).(pulumi.StringOutput)
	}

	// Don't default storage class name -> will select the default one
	// on the K8s cluster.
	args.storageClassName = pulumi.String(defaultStorageClassName).ToStringOutput()
	if args.StorageClassName != nil {
		args.storageClassName = args.StorageClassName.ToStringOutput().ApplyT(func(scm string) string {
			if scm == "" {
				return defaultStorageClassName
			}
			return scm
		}).(pulumi.StringOutput)
	}

	// Default storage size to 50M
	args.storageSize = pulumi.String(defaultStorageSize).ToStringOutput()
	if args.StorageSize != nil {
		args.storageSize = args.StorageSize.ToStringOutput().ApplyT(func(size string) string {
			if size == "" {
				return defaultStorageSize
			}
			return size
		}).(pulumi.StringOutput)
	}

	// Define private registry if any
	args.registry = pulumi.String("").ToStringOutput()
	if args.Registry != nil {
		args.registry = args.Registry.ToStringOutput().ApplyT(func(in string) string {
			if !strings.HasSuffix(in, "/") {
				in += "/"
			}
			return in
		}).(pulumi.StringOutput)
	}

	// Default PVC access modes
	if args.PVCAccessModes == nil {
		args.pvcAccessModes = pulumi.ToStringArray([]string{
			"ReadWriteMany",
		}).ToStringArrayOutput()
	} else {
		args.pvcAccessModes = args.PVCAccessModes.ToStringArrayOutput().ApplyT(func(slc []string) []string {
			if len(slc) == 0 {
				return []string{"ReadWriteMany"}
			}
			return slc
		}).(pulumi.StringArrayOutput)
	}

	return args
}

func (renv *RomeoEnvironment) provision(
	ctx *pulumi.Context,
	name string,
	args *RomeoEnvironmentArgs,
	opts ...pulumi.ResourceOption,
) (err error) {
	// Generate unique (random enough) PVC name
	renv.randName, err = random.NewRandomString(ctx, "romeo-name-"+name, &random.RandomStringArgs{
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
	renv.pvc, err = corev1.NewPersistentVolumeClaim(ctx, "romeo-pvc-"+name, &corev1.PersistentVolumeClaimArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String(name),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				"instance":                    renv.randName.Result,
			},
			Name: renv.randName.Result,
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
				ClaimName: renv.pvc.Metadata.Name().Elem(),
			},
		},
	}
	if args.ClaimName != nil {
		fmt.Println("Deploying with a claim name thus coverage exports")

		// If coverage is turned on, export coverages in a random directory
		// that is different from coverdir (ensure no collision).
		renv.coverRand, err = random.NewRandomString(ctx, "cover-rand-"+name, &random.RandomStringArgs{
			Length:  pulumi.Int(16),
			Lower:   pulumi.BoolPtr(true),
			Numeric: pulumi.BoolPtr(false),
			Special: pulumi.BoolPtr(false),
			Upper:   pulumi.BoolPtr(false),
		}, opts...)
		if err != nil {
			return
		}
		path := renv.coverRand.Result.ApplyT(func(rand string) string {
			return fmt.Sprintf("/tmp/%s", rand)
		}).(pulumi.StringOutput)

		envs = append(envs, corev1.EnvVarArgs{
			Name:  pulumi.String("GOCOVERDIR"),
			Value: path,
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMountArgs{
			Name:      pulumi.String("coverages"),
			MountPath: path,
		})
		volumes = append(volumes, corev1.VolumeArgs{
			Name: pulumi.String("coverages"),
			PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSourceArgs{
				ClaimName: args.ClaimName,
			},
		})
	}
	renv.dep, err = appsv1.NewDeployment(ctx, "romeo-dep-"+name, &appsv1.DeploymentArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name":      pulumi.String("romeo"),
				"app.kubernetes.io/version":   args.tag,
				"app.kubernetes.io/component": pulumi.String(name),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				"instance":                    renv.randName.Result,
			},
		},
		Spec: appsv1.DeploymentSpecArgs{
			Selector: metav1.LabelSelectorArgs{
				MatchLabels: pulumi.StringMap{
					"app.kubernetes.io/name":      pulumi.String("romeo"),
					"app.kubernetes.io/version":   args.tag,
					"app.kubernetes.io/component": pulumi.String(name),
					"app.kubernetes.io/part-of":   pulumi.String("romeo"),
					"instance":                    renv.randName.Result,
				},
			},
			Replicas: pulumi.Int(1),
			Template: corev1.PodTemplateSpecArgs{
				Metadata: metav1.ObjectMetaArgs{
					Namespace: args.Namespace,
					Labels: pulumi.StringMap{
						"app.kubernetes.io/name":      pulumi.String("romeo"),
						"app.kubernetes.io/version":   args.tag,
						"app.kubernetes.io/component": pulumi.String(name),
						"app.kubernetes.io/part-of":   pulumi.String("romeo"),
						"instance":                    renv.randName.Result,
					},
				},
				Spec: corev1.PodSpecArgs{
					Containers: corev1.ContainerArray{
						corev1.ContainerArgs{
							Name:            pulumi.String("romeo"),
							Image:           pulumi.Sprintf("%sctferio/romeo:%s", args.registry, args.tag),
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
	renv.svc, err = corev1.NewService(ctx, "romeo-svc-"+name, &corev1.ServiceArgs{
		Metadata: metav1.ObjectMetaArgs{
			Namespace: args.Namespace,
			Labels: pulumi.StringMap{
				"app.kubernetes.io/component": pulumi.String(name),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				"instance":                    renv.randName.Result,
			},
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("NodePort"),
			Selector: pulumi.StringMap{
				"app.kubernetes.io/name":      pulumi.String("romeo"),
				"app.kubernetes.io/version":   args.tag,
				"app.kubernetes.io/component": pulumi.String(name),
				"app.kubernetes.io/part-of":   pulumi.String("romeo"),
				"instance":                    renv.randName.Result,
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

func (renv *RomeoEnvironment) outputs(ctx *pulumi.Context) error {
	renv.Namespace = renv.dep.Metadata.Namespace().Elem()
	renv.ClaimName = renv.pvc.Metadata.Name().Elem()
	renv.Port = renv.svc.Spec.Ports().Index(pulumi.Int(0)).NodePort().Elem()

	return ctx.RegisterResourceOutputs(renv, pulumi.Map{
		"namespace":  renv.Namespace,
		"claim-name": renv.ClaimName,
		"port":       renv.Port,
	})
}
