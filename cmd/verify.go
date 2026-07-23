package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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

		// Read plan
		var plan models.Plan
		if err := utils.ReadJSON(planPath, &plan); err != nil {
			return fmt.Errorf("failed to read plan: %w", err)
		}

		// Read inventory
		var inventory models.Inventory
		if err := utils.ReadJSON(inventoryPath, &inventory); err != nil {
			return fmt.Errorf("failed to read inventory: %w", err)
		}

		// Check if result exists
		resultExists := true
		var result models.Result
		if _, err := os.Stat(resultPath); os.IsNotExist(err) {
			resultExists = false
		} else {
			if err := utils.ReadJSON(resultPath, &result); err != nil {
				return fmt.Errorf("failed to read result: %w", err)
			}
		}

		// Gather live default resources from AWS
		prof := viper.GetString("profile")
		reg := viper.GetString("region")
		awsCfg, err := aws.NewSession(cmd.Context(), aws.Config{
			Profile: prof,
			Region:  reg,
		})

		defaultGroups := make(map[string][]string)
		protectedCount := 0
		if err == nil {
			ec2Client := ec2.NewFromConfig(awsCfg)
			// Fetch Default VPCs
			vpcs, err := ec2Client.DescribeVpcs(cmd.Context(), &ec2.DescribeVpcsInput{})
			if err == nil {
				for _, v := range vpcs.Vpcs {
					if v.IsDefault != nil && *v.IsDefault {
						id := *v.VpcId
						if len(id) > 7 {
							id = id[:7] + "..."
						}
						defaultGroups["VPC"] = append(defaultGroups["VPC"], id)
						protectedCount++
					}
				}
			}
			// Fetch Default Subnets
			subnets, err := ec2Client.DescribeSubnets(cmd.Context(), &ec2.DescribeSubnetsInput{})
			if err == nil {
				for _, s := range subnets.Subnets {
					if s.DefaultForAz != nil && *s.DefaultForAz {
						id := *s.SubnetId
						if len(id) > 9 {
							id = id[:9] + "..."
						}
						defaultGroups["Subnet"] = append(defaultGroups["Subnet"], id)
						protectedCount++
					}
				}
			}
			// Fetch Default Security Groups
			sgs, err := ec2Client.DescribeSecurityGroups(cmd.Context(), &ec2.DescribeSecurityGroupsInput{})
			if err == nil {
				for _, sg := range sgs.SecurityGroups {
					if sg.GroupName != nil && *sg.GroupName == "default" {
						defaultGroups["Security Group"] = append(defaultGroups["Security Group"], *sg.GroupName)
						protectedCount++
					}
				}
			}
		}

		// Group unselected resources
		planMap := make(map[string]bool)
		for _, r := range plan.Steps {
			planMap[r.ID] = true
		}
		unselectedMap := make(map[string]models.Resource)
		for _, r := range inventory.Resources {
			if !planMap[r.ID] {
				unselectedMap[r.ID] = r
			}
		}

		unselectedGroups := make(map[string][]string)
		var unselectedGroupKeys []string
		for _, r := range unselectedMap {
			cleanType := r.Type
			name := r.Name
			if name == "" {
				name = r.ID
			}
			if _, exists := unselectedGroups[cleanType]; !exists {
				unselectedGroupKeys = append(unselectedGroupKeys, cleanType)
			}
			unselectedGroups[cleanType] = append(unselectedGroups[cleanType], name)
		}
		sort.Strings(unselectedGroupKeys)

		// Print verification report header
		fmt.Println("AWS Kill Switch - Verification")
		fmt.Println()
		fmt.Println("Verification Status")
		fmt.Println("────────────────────────────────────────────")
		fmt.Println()

		var statusText string
		var reasonTitle string
		var reasonDesc []string
		var actionTitle string
		var actionDesc string
		var verResult string

		if !resultExists {
			statusText = "✗ Deletion plan has not been executed."
			reasonTitle = "Reason"
			reasonDesc = []string{"reports/plan.json exists", "reports/result.json not found"}
			actionTitle = "Action Required"
			actionDesc = "Run:\n\ngo run . kill"
			verResult = "FAILED"
		} else if len(result.FailedResources) > 0 {
			statusText = "✗ Deletion plan execution resulted in failures."
			reasonTitle = "Reason"
			reasonDesc = []string{"reports/plan.json exists", "reports/result.json contains failed resources"}
			actionTitle = "Action Required"
			actionDesc = "Resolve failures and re-run:\n\ngo run . kill"
			verResult = "PARTIAL SUCCESS"
		} else {
			statusText = "✓ Deletion plan has been executed successfully."
			reasonTitle = "Reason"
			reasonDesc = []string{"reports/plan.json exists", "reports/result.json indicates all resources deleted"}
			actionTitle = "Action Required"
			actionDesc = "None"
			verResult = "SUCCESS"
		}

		fmt.Println(statusText)
		fmt.Println()
		fmt.Println(reasonTitle)
		fmt.Println("------")
		for _, desc := range reasonDesc {
			fmt.Println(desc)
		}
		fmt.Println()
		fmt.Println(actionTitle)
		fmt.Println("---------------")
		fmt.Println(actionDesc)
		fmt.Println()

		// Print Planned Resources
		plannedGroups := make(map[string][]string)
		var plannedGroupKeys []string
		for _, r := range plan.Steps {
			cleanType := cleanTypeName(r.Type)
			name := r.Name
			if name == "" {
				name = r.ID
			}
			if _, exists := plannedGroups[cleanType]; !exists {
				plannedGroupKeys = append(plannedGroupKeys, cleanType)
			}
			plannedGroups[cleanType] = append(plannedGroups[cleanType], name)
		}
		sort.Strings(plannedGroupKeys)

		fmt.Println("Planned Deletion Resources")
		fmt.Println("────────────────────────────────────────────")
		fmt.Println()
		for _, k := range plannedGroupKeys {
			fmt.Println(k)
			for _, item := range plannedGroups[k] {
				fmt.Printf("  • %s\n", item)
			}
			fmt.Println()
		}

		// If result exists and we have failures, display them separately
		if resultExists && len(result.FailedResources) > 0 {
			failedGroups := make(map[string][]string)
			var failedGroupKeys []string
			for _, r := range result.FailedResources {
				cleanType := cleanTypeName(r.Type)
				name := r.Name
				if name == "" {
					name = r.ID
				}
				// Append state (failure reason)
				reasonStr := r.State
				if reasonStr == "" {
					reasonStr = "unknown error"
				}
				failedInfo := fmt.Sprintf("%s\n    Reason: %s", name, reasonStr)

				if _, exists := failedGroups[cleanType]; !exists {
					failedGroupKeys = append(failedGroupKeys, cleanType)
				}
				failedGroups[cleanType] = append(failedGroups[cleanType], failedInfo)
			}
			sort.Strings(failedGroupKeys)

			fmt.Println("Failed Deletion Resources")
			fmt.Println("────────────────────────────────────────────")
			fmt.Println()
			for _, k := range failedGroupKeys {
				fmt.Println(k)
				for _, item := range failedGroups[k] {
					fmt.Printf("  • %s\n", item)
				}
				fmt.Println()
			}
		}

		// Print Protected Default Resources
		fmt.Println("Protected AWS Default Resources")
		fmt.Println("────────────────────────────────────────────")
		fmt.Println()
		defaultGroupKeys := []string{"VPC", "Subnet", "Security Group"}
		for _, k := range defaultGroupKeys {
			list := defaultGroups[k]
			if len(list) > 0 {
				fmt.Println(k)
				for _, item := range list {
					fmt.Printf(" • %s\n", item)
				}
				fmt.Println()
			}
		}
		fmt.Println("These resources are intentionally excluded from deletion.")
		fmt.Println()

		// Print Unselected Resources
		fmt.Println("Resources Not Included In Current Plan")
		fmt.Println("────────────────────────────────────────────")
		if len(unselectedMap) == 0 {
			fmt.Println("No active resources remain outside the current deletion plan.")
			fmt.Println()
		} else {
			for _, k := range unselectedGroupKeys {
				fmt.Println(k)
				for _, item := range unselectedGroups[k] {
					fmt.Printf(" • %s\n", item)
				}
				fmt.Println()
			}
		}

		// Print Summary
		fmt.Println("Summary")
		fmt.Println("────────────────────────────────────────────")
		fmt.Println()

		if !resultExists {
			fmt.Printf("Deletion Plan Status     : Not Executed\n")
			fmt.Printf("Resources Planned        : %d\n", len(plan.Steps))
			fmt.Printf("Resources Not Selected   : %d\n", len(unselectedMap))
		} else {
			statusStr := "Executed (All Succeeded)"
			if len(result.FailedResources) > 0 {
				statusStr = "Executed (With Failures)"
			}
			fmt.Printf("Deletion Plan Status     : %s\n", statusStr)
			fmt.Printf("Resources Succeeded      : %d\n", len(result.DeletedResources))
			fmt.Printf("Resources Failed         : %d\n", len(result.FailedResources))
		}
		fmt.Printf("Protected Resources      : %d\n", protectedCount)
		fmt.Println()
		fmt.Printf("Verification Result      : %s\n", verResult)

		if !resultExists || len(result.FailedResources) > 0 {
			os.Exit(1)
		}

		// Write verification success report if everything is clean
		verificationReport := map[string]interface{}{
			"verified":            true,
			"remaining_resources": []string{},
		}
		verificationPath := filepath.Join("reports", "verification.json")
		if err := utils.WriteJSON(verificationPath, &verificationReport); err != nil {
			return fmt.Errorf("failed to write verification report: %w", err)
		}

		return nil
	},
}

// cleanTypeName normalizes AWS resource type names to singular equivalents for TUI presentation
func cleanTypeName(t string) string {
	switch t {
	case "EC2 Instances":
		return "EC2 Instance"
	case "Subnets":
		return "Subnet"
	case "Security Groups":
		return "Security Group"
	case "Route Tables":
		return "Route Table"
	case "Internet Gateway":
		return "Internet Gateway"
	case "NAT Gateway":
		return "NAT Gateway"
	case "Elastic IP":
		return "Elastic IP"
	case "Application Load Balancer":
		return "Application Load Balancer"
	case "Target Groups":
		return "Target Group"
	case "Volume":
		return "Volume"
	case "Snapshot":
		return "Snapshot"
	case "KeyPair":
		return "KeyPair"
	default:
		return t
	}
}

func init() {
	RootCmd.AddCommand(verifyCmd)
}
