package aws

import (
	"polyqube/pkg/aws/k3s"
	create "polyqube/pkg/cli/commands"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateClusters(ctx *pulumi.Context) error {
	clusters, err := create.LoadClusters("aws")
	if err != nil {
		return err
	}

	return createClusters(ctx, clusters, false)
}

func CreateDevClusters(ctx *pulumi.Context) error {
	clusters, err := create.LoadClusters("aws_dev")
	if err != nil {
		return err
	}

	return createClusters(ctx, clusters, true)
}

func createClusters(ctx *pulumi.Context, clusters create.Config, isDev bool) error {
	for _, clusters := range clusters.Regions {
		for cluster, clusterDef := range clusters.Clusters {
			clusterName := cluster
			workerCount := clusterDef.WorkerCount
			controlPlaneCount := clusterDef.ControlPlaneCount
			// templateId := clusterDef.TemplateId
			// TODO: Remove hardcoding of AMI
			templateId := "ami-058ce68c84d0e4c93"

			clusterOpts, err := k3s.
				NewClusterOpts().
				Name(clusterName).
				Ami(templateId).
				ControlPlaneNodes(controlPlaneCount).
				WorkerNodes(workerCount).
				Build()
			if err != nil {
				return err
			}

			cluster := k3s.NewCluster(clusterOpts)
			if err := cluster.Create(ctx, isDev); err != nil {
				return err
			}
		}
	}

	return nil
}
