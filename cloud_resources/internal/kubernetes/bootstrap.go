package kubernetes

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	utils "polyqube/internal/k3s"
	apps "polyqube/internal/kubernetes/argocd"
)

type bootstrapper struct {
	ctx          *pulumi.Context
	clusterName  string
	kubernetes   *kubernetes.Provider
	clustersetup *utils.ClusterSetup
	dependencies []pulumi.Resource
}

// Initialize a new K3S manifest bootstrapper
func NewBootstrapper(ctx *pulumi.Context, provider *kubernetes.Provider, clusterName string, clustersetup *utils.ClusterSetup, dependencies []pulumi.Resource) *bootstrapper {
	return &bootstrapper{
		ctx:          ctx,
		clusterName:  clusterName,
		kubernetes:   provider,
		clustersetup: clustersetup,
		dependencies: dependencies,
	}
}

func (b *bootstrapper) Setup() error {
	appManager := apps.NewAppManager(b.ctx, b.clusterName, b.kubernetes)
	if err := appManager.DeployAllApps(); err != nil {
		return err
	}
	return nil
}
