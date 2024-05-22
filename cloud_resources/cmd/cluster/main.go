package main

import (
	"errors"
	"fmt"
	"os"

	create "polyqube/pkg/cli/commands"
	"polyqube/pkg/cli/utils"

	"github.com/spf13/cobra"
)

var (
	ErrInvalidPlatform = errors.New("invalid cloud platform")
	ErrInvalidRegion   = errors.New("invalid region")
)

func validatePlatform(val string) error {
	if err := utils.IsWithinStringSlice(val, utils.Platforms); err != nil {
		return ErrInvalidPlatform
	}
	return nil
}

func validateRegion(val string) error {
	return utils.ValidAWSRegion(val)
}

func validate(platform, region string) error {
	if err := validatePlatform(platform); err != nil {
		return ErrInvalidPlatform
	}

	if err := validateRegion(region); err != nil {
		return ErrInvalidRegion
	}

	return nil
}

func main() {
	var alias string

	var rootCmd = &cobra.Command{
		Use:   "create-cluster",
		Short: "Create a new cluster definition",
		Run: func(cmd *cobra.Command, args []string) {
			name, err := utils.GenerateUniqueString(12)
			if err != nil {
				fmt.Printf("Failed to generate unique cluster name: %s\n", err)
				os.Exit(1)
			}

			platform, err := utils.PromptPlatform()
			if err != nil {
				fmt.Printf("Error selecting platform: %s\n", err)
				os.Exit(1)
			}

			region, err := utils.PromptRegionBasedOnPlatform(platform)
			if err != nil {
				fmt.Printf("Error selecting region: %s\n", err)
				os.Exit(1)
			}

			workerNodeCount, err := utils.PromptNodeCount("worker")
			if err != nil {
				fmt.Printf("Error selecting worker node count: %s\n", err)
				os.Exit(1)
			}

			controlPlaneCount, err := utils.PromptNodeCount("control plane")
			if err != nil {
				fmt.Printf("Error selecting control plane node count: %s\n", err)
				os.Exit(1)
			}

			if err := validate(platform, region); err != nil {
				fmt.Printf("Error validating cluster definition: %s\n", err)
				os.Exit(1)
			}

			if err := create.CreateCluster(platform, region, name, workerNodeCount, controlPlaneCount); err != nil {
				fmt.Printf("Error creating cluster definition: %s\n", err)
				os.Exit(1)
			}

			fmt.Println("Cluster definition created successfully")
			fmt.Printf("Cluster name: %s\n", name)
			fmt.Printf("Cluster alias: %s\n", alias)
		},
	}

	// rootCmd.Flags().StringVarP(&alias, "alias", "n", "", "Alias of the cluster")
	// rootCmd.Flags().StringVarP(&region, "region", "r", "", "Region of the Pulumi stack")
	// rootCmd.Flags().StringVarP(&platform, "platform", "p", "", "Cloud platform")
	// rootCmd.MarkFlagRequired("region")
	// rootCmd.MarkFlagRequired("platform")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
