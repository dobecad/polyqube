package main

import (
	"os"
	"polyqube/clusters/aws"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateResources() error {
	isProd := os.Getenv("PROD")
	if isProd == "TRUE" {
		pulumi.Run(func(ctx *pulumi.Context) error {
			return createResources(ctx)
		})
	} else {
		pulumi.Run(func(ctx *pulumi.Context) error {
			return createDevResources(ctx)
		})
	}

	return nil
}

func createResources(ctx *pulumi.Context) error {
	if err := aws.CreateClusters(ctx); err != nil {
		return err
	}

	return nil
}

func createDevResources(ctx *pulumi.Context) error {
	if err := aws.CreateDevClusters(ctx); err != nil {
		return err
	}

	return nil
}
