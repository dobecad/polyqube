package apps

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type AppManager struct {
	ctx         *pulumi.Context
	clusterName string
	kubernetes  *kubernetes.Provider
}

func NewAppManager(ctx *pulumi.Context, clusterName string, kubernetes *kubernetes.Provider) *AppManager {
	return &AppManager{
		ctx:         ctx,
		clusterName: clusterName,
		kubernetes:  kubernetes,
	}
}

func (a *AppManager) CreateNamespace(name string, dependsOn []pulumi.Resource) (*v1.Namespace, error) {
	resourceName := fmt.Sprintf("%s-%s-namespace", a.clusterName, name)
	ns, err := v1.NewNamespace(a.ctx, resourceName, &v1.NamespaceArgs{
		ApiVersion: pulumi.String("v1"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String(name),
		},
	}, pulumi.DependsOn(dependsOn), pulumi.Provider(a.kubernetes))

	if err != nil {
		return nil, err
	}

	return ns, nil
}

func (a *AppManager) CreateArgoGitRepoSecret(dependsOn []pulumi.Resource) (*v1.Secret, error) {
	conf := config.New(a.ctx, "")
	argocdRepoCreds := conf.Get("argocd-repo-key")
	if argocdRepoCreds == "" {
		panic("missing argocd repo key")
	}

	secret, err := v1.NewSecret(a.ctx, fmt.Sprintf("%s-argocd-repo-key", a.clusterName), &v1.SecretArgs{
		ApiVersion: pulumi.String("v1"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("private-repo"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"argocd.argoproj.io/secret-type": pulumi.String("repository"),
			},
		},
		StringData: pulumi.StringMap{
			"type":          pulumi.String("git"),
			"url":           pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
			"sshPrivateKey": pulumi.Sprintf("%s", argocdRepoCreds),
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func (a *AppManager) DeployArgocd(dependsOn []pulumi.Resource) (*helm.Release, error) {
	chart, err := helm.NewRelease(a.ctx, fmt.Sprintf("%s-argocd", a.clusterName), &helm.ReleaseArgs{
		Chart:   pulumi.String("argo-cd"),
		Version: pulumi.String("6.0.14"),
		Name:    pulumi.String("argocd"),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://argoproj.github.io/argo-helm"),
		},
		Atomic: pulumi.Bool(true),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"nodeSelector": pulumi.Map{
					"cloud-worker": pulumi.String("true"),
				},
			},
		},
		Namespace: pulumi.String("argocd"),
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func (a *AppManager) DeployMonitoring(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-monitoring", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("monitoring"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("monitoring"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/k8s-monitoring"),
					"helm": pulumi.Map{
						"ignoreMissingValueFiles": pulumi.Bool(true),
						"parameters": pulumi.MapArray{
							pulumi.Map{
								"name":  pulumi.String("cluster.name"),
								"value": pulumi.String(a.clusterName),
							},
						},
					},
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("argocd"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployNvidiaRuntime(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-nvidia-rt-device", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("nvidia-runtime-and-device-plugin"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("nvidia-runtime-and-device"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/nvidia"),
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("kube-system"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployNodeFeatureDiscovery(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-nfd", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("node-feature-discovery"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("node-feature-discovery"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/nfd/node-feature-discovery"),
					"helm": pulumi.Map{
						"ignoreMissingValueFiles": pulumi.Bool(true),
					},
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("kube-system"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployGpuMonitoring(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-gpu-monitoring", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("gpu-monitoring"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("gpu-monitoring"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/dcgm-exporter"),
					"helm": pulumi.Map{
						"ignoreMissingValueFiles": pulumi.Bool(true),
					},
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("kube-system"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployGpuFeatureDiscovery(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-gpu-fd", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("gpu-feature-discovery"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("gpu-feature-discovery"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/gfd"),
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("kube-system"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployNvidiaGpuOperator(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-nvidia-gpu-operator", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("nvidia-gpu-operator"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("nvidia-gpu-operator"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/gpu-operator"),
					"helm": pulumi.Map{
						"ignoreMissingValueFiles": pulumi.Bool(true),
					},
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("gpu-operator"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (a *AppManager) DeployTraefik(dependsOn []pulumi.Resource) (*apiextensions.CustomResource, error) {
	resource, err := apiextensions.NewCustomResource(a.ctx, fmt.Sprintf("%s-traefik", a.clusterName), &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("argoproj.io/v1alpha1"),
		Kind:       pulumi.String("Application"),
		Metadata: metav1.ObjectMetaArgs{
			Name:      pulumi.String("traefik"),
			Namespace: pulumi.String("argocd"),
			Labels: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("traefik"),
			},
			Finalizers: pulumi.StringArray{
				pulumi.String("resources-finalizer.argocd.argoproj.io"),
			},
		},
		OtherFields: kubernetes.UntypedArgs{
			"spec": pulumi.Map{
				"project": pulumi.String("default"),
				"source": pulumi.Map{
					"repoURL":        pulumi.String("git@github.com:dobecad/polyqube-cluster-manifests.git"),
					"targetRevision": pulumi.String("main"),
					"path":           pulumi.String("dev/components/manifests/traefik"),
					"helm": pulumi.Map{
						"ignoreMissingValueFiles": pulumi.Bool(true),
					},
				},
				"destination": pulumi.Map{
					"server":    pulumi.String("https://kubernetes.default.svc"),
					"namespace": pulumi.String("kube-system"),
				},
				"syncPolicy": pulumi.Map{
					"automated": pulumi.Map{
						"prune":      pulumi.Bool(true),
						"selfHeal":   pulumi.Bool(true),
						"allowEmpty": pulumi.Bool(true),
					},
					"syncOptions": pulumi.StringArray{
						pulumi.String("CreateNamespace=true"),
						pulumi.String("PrunePropagationPolicy=true"),
						pulumi.String("ApplyOutOfSyncOnly=true"),
						pulumi.String("RespectIgnoreDifferences=true"),
					},
				},
			},
		},
	}, pulumi.Provider(a.kubernetes), pulumi.DependsOn(dependsOn))
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// Deploy all ArgoCD apps
func (a *AppManager) DeployAllApps() error {
	ns, err := a.CreateNamespace("argocd", []pulumi.Resource{})
	if err != nil {
		return err
	}

	repoSecret, err := a.CreateArgoGitRepoSecret([]pulumi.Resource{ns})
	if err != nil {
		return err
	}

	argocd, err := a.DeployArgocd([]pulumi.Resource{ns, repoSecret})
	if err != nil {
		return err
	}

	if _, err := a.DeployMonitoring([]pulumi.Resource{ns, repoSecret, argocd}); err != nil {
		return err
	}

	nvidiaRuntime, err := a.DeployNvidiaRuntime([]pulumi.Resource{ns, repoSecret, argocd})
	if err != nil {
		return err
	}

	gfd, err := a.DeployGpuFeatureDiscovery([]pulumi.Resource{ns, repoSecret, argocd, nvidiaRuntime})
	if err != nil {
		return err
	}

	gpuMon, err := a.DeployGpuMonitoring([]pulumi.Resource{ns, repoSecret, argocd, nvidiaRuntime})
	if err != nil {
		return err
	}

	nfd, err := a.DeployNodeFeatureDiscovery([]pulumi.Resource{ns, repoSecret, argocd, nvidiaRuntime})
	if err != nil {
		return err
	}

	if _, err := a.DeployNvidiaGpuOperator([]pulumi.Resource{ns, repoSecret, argocd, nvidiaRuntime, gfd, gpuMon, nfd}); err != nil {
		return err
	}

	if _, err := a.DeployTraefik([]pulumi.Resource{ns, repoSecret, argocd}); err != nil {
		return err
	}

	return nil
}
