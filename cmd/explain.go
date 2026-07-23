package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Troubleshoot and explain why planned resource deletions failed",
	Long:  `Analyzes reports/result.json to identify failed resources, explain the AWS dependency blockages, and provide fixes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		resultPath := filepath.Join("reports", "result.json")

		if _, err := os.Stat(resultPath); os.IsNotExist(err) {
			fmt.Printf("Error: No execution results found at %s. Please run 'aws-kill kill' first.\n", resultPath)
			os.Exit(1)
		}

		var result models.Result
		if err := utils.ReadJSON(resultPath, &result); err != nil {
			return fmt.Errorf("failed to read result file: %w", err)
		}

		if len(result.FailedResources) == 0 {
			fmt.Println("No execution failures found. All planned resources were successfully terminated!")
			return nil
		}

		fmt.Println("AWS Kill Switch - Troubleshooting Guide")
		fmt.Println("────────────────────────────────────────────")

		for _, r := range result.FailedResources {
			cleanType := cleanExplainTypeName(r.Type)
			name := r.Name
			if name == "" {
				name = r.ID
			}

			rawErr := r.State // State holds the failure reason error string
			why := "AWS API returned a raw deletion error."
			fix := "Check the AWS console for active dependencies or retry execution using 'go run . kill --retry'."

			errLower := strings.ToLower(rawErr)

			if strings.Contains(errLower, "bucketnotempty") {
				why = "The S3 bucket contains files, active object versions, or delete markers."
				fix = fmt.Sprintf("Run: aws s3 rb s3://%s --force\n            Or empty the bucket contents in the AWS Console.", r.ID)
			} else if strings.Contains(errLower, "dependencyviolation") {
				if r.Type == "Security Groups" {
					why = "This security group is cross-referenced by another security group's ingress/egress rule, or attached to an active Network Interface (ENI)."
					fix = "Revoke ingress/egress rules that cross-reference this group, or wait for dependent ENIs to detach."
				} else {
					why = "The resource is still linked or in use by another active service."
					fix = "Identify and terminate the parent resource (e.g. EC2 instance, ALB) first, then retry."
				}
			} else if strings.Contains(errLower, "invalidifmatchversion") || strings.Contains(errLower, "invalidstate") {
				if r.Type == "CloudFront" {
					why = "CloudFront distributions must be disabled and fully deployed globally before they can be deleted."
					fix = fmt.Sprintf("Disable the distribution %s in the AWS Console, wait for its status to be 'Deployed', and then delete.", r.ID)
				}
			} else if strings.Contains(errLower, "invalidnetworkinterfaceid.notfound") || strings.Contains(errLower, "invalidvolume.notfound") || strings.Contains(errLower, "invalidallocationid.notfound") {
				why = "The resource ID was not found in AWS. It was likely already deleted by a previous step or manually."
				fix = "No action required. Run 'go run . verify' to confirm its deletion status."
			}

			fmt.Printf("[%s] %s\n", cleanType, name)
			fmt.Printf("  Status  : Failed\n")
			fmt.Printf("  Why     : %s\n", why)
			fmt.Printf("  Fix     : %s\n", fix)
			fmt.Println("────────────────────────────────────────────")
		}

		return nil
	},
}

func cleanExplainTypeName(t string) string {
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
		return "EBS Volume"
	case "Snapshot":
		return "Snapshot"
	case "KeyPair":
		return "Key Pair"
	case "CloudFront":
		return "CloudFront Distribution"
	case "S3":
		return "S3 Bucket"
	case "RDS":
		return "RDS Instance"
	default:
		return t
	}
}

func init() {
	RootCmd.AddCommand(explainCmd)
}
