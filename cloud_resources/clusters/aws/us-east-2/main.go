package clusters

import (
	"polyqube/pkg/aws/k3s"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func DevCluster(ctx *pulumi.Context) error {
	clusterName := "dev"
	controlPlaneCount := uint8(3)
	workerCount := uint8(2)
	// Free tier Ubuntu 22.04 ami id for us-east-2
	// ami := "ami-0f5daaa3a7fb3378b"
	ami := "ami-058ce68c84d0e4c93"

	clusterOpts, err := k3s.
		NewClusterOpts().
		Name(clusterName).
		Ami(ami).
		ControlPlaneNodes(controlPlaneCount).
		WorkerNodes(workerCount).
		Build()
	if err != nil {
		return err
	}

	cluster := k3s.NewCluster(clusterOpts)
	if err := cluster.Create(ctx, false); err != nil {
		return err
	}

	return nil
}

// Create all the clusters associated with AWS us-east-2 region.
//
// Clusters will only be created if the currently selected stack
// matches the cluster name. This allows us to be selective
// in what cluster is being created/updated, while still
// aggregating the cluster creation for a region in one function
func CreateClusters(ctx *pulumi.Context) error {
	clusters := []func(*pulumi.Context) error{DevCluster}

	for _, launchFunc := range clusters {
		if err := launchFunc(ctx); err != nil {
			return err
		}
	}

	return nil
}
