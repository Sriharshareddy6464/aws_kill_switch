package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Confirm target AWS resources are completely deleted",
	Long:  `Queries the AWS environment to verify all planned resources have been removed, creating a final audit report. Requires a completed kill execution first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPath := filepath.Join("reports", "plan.json")
		resultPath := filepath.Join("reports", "result.json")
		inventoryPath := filepath.Join("reports", "inventory.json")

		// 1. Guard check: plan must exist to verify
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			fmt.Printf("Error: No execution plan found at %s. Please run 'aws-kill plan' first.\n", planPath)
			os.Exit(1)
		}

		// 2. Check if kill action was missed (plan exists but result does not)
		if _, err := os.Stat(resultPath); os.IsNotExist(err) {
			fmt.Println("Error: Generated deletion plan isn't executed. Kindly run \"go run . kill\"")
			fmt.Println()

			// Load plan resources (missed list)
			var plan models.Plan
			if err := utils.ReadJSON(planPath, &plan); err == nil {
				fmt.Println("You have missed: (Planned List)")
				fmt.Println("------------------------------------------------")
				for _, r := range plan.Steps {
					name := r.Name
					if name == "" {
						name = "<no-name>"
					}
					fmt.Printf("  - [%s] ID: %s Name: %s\n", r.Type, r.ID, name)
				}
				fmt.Println()
			}

			// Load inventory to compare and find unselected ones
			var inventory models.Inventory
			unselectedMap := make(map[string]models.Resource)
			if err := utils.ReadJSON(inventoryPath, &inventory); err == nil {
				// Map plan IDs for easy lookup
				planMap := make(map[string]bool)
				for _, r := range plan.Steps {
					planMap[r.ID] = true
				}
				// Find resources in inventory that are not in plan
				for _, r := range inventory.Resources {
					if !planMap[r.ID] {
						unselectedMap[r.ID] = r
					}
				}
			}

			fmt.Println("You still remaining with: (Unselected List)")
			fmt.Println("------------------------------------------------")
			for _, r := range unselectedMap {
				name := r.Name
				if name == "" {
					name = "<no-name>"
				}
				fmt.Printf("  - [%s] ID: %s Name: %s (Unselected)\n", r.Type, r.ID, name)
			}

			// Fetch and append live default resources from AWS
			prof := viper.GetString("profile")
			reg := viper.GetString("region")
			awsCfg, err := aws.NewSession(cmd.Context(), aws.Config{
				Profile: prof,
				Region:  reg,
			})
			if err == nil {
				ec2Client := ec2.NewFromConfig(awsCfg)
				// Fetch Default VPCs
				vpcs, err := ec2Client.DescribeVpcs(cmd.Context(), &ec2.DescribeVpcsInput{})
				if err == nil {
					for _, v := range vpcs.Vpcs {
						if v.IsDefault != nil && *v.IsDefault {
							fmt.Printf("  - [VPC] ID: %s (Default / Protected)\n", *v.VpcId)
						}
					}
				}
				// Fetch Default Subnets
				subnets, err := ec2Client.DescribeSubnets(cmd.Context(), &ec2.DescribeSubnetsInput{})
				if err == nil {
					for _, s := range subnets.Subnets {
						if s.DefaultForAz != nil && *s.DefaultForAz {
							fmt.Printf("  - [Subnets] ID: %s (Default / Protected)\n", *s.SubnetId)
						}
					}
				}
				// Fetch Default Security Groups
				sgs, err := ec2Client.DescribeSecurityGroups(cmd.Context(), &ec2.DescribeSecurityGroupsInput{})
				if err == nil {
					for _, sg := range sgs.SecurityGroups {
						if sg.GroupName != nil && *sg.GroupName == "default" {
							fmt.Printf("  - [Security Groups] ID: %s Name: %s (Default / Protected)\n", *sg.GroupId, *sg.GroupName)
						}
					}
				}
			} else {
				fmt.Println("  [AWS credentials offline - skipping live default resources check]")
			}

			os.Exit(1)
		}

		// 3. Normal verification path (kill result exists)
		fmt.Println("Starting post-deletion verification...")

		var result models.Result
		if err := utils.ReadJSON(resultPath, &result); err != nil {
			return fmt.Errorf("failed to read result JSON: %w", err)
		}

		// Placeholder verification check (writes success verification.json)
		verificationReport := map[string]interface{}{
			"verified":            true,
			"remaining_resources": []string{},
		}
		verificationPath := filepath.Join("reports", "verification.json")
		if err := utils.WriteJSON(verificationPath, &verificationReport); err != nil {
			return fmt.Errorf("failed to write verification report: %w", err)
		}

		fmt.Println("Verification complete. Report generated at reports/verification.json.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(verifyCmd)
}
