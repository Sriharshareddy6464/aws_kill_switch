package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

type verificationReport struct {
	Status    string `json:"status"`
	Resources []struct {
		Service      string `json:"service"`
		ID           string `json:"id"`
		Name         string `json:"name"`
		Verification string `json:"verification"`
		Reason       string `json:"reason"`
	} `json:"resources"`
}

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Troubleshoot and explain why planned resource deletions failed",
	Long:  `Analyzes reports/verification.json to identify remaining active resources, explain the AWS dependency blockages, and provide fixes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verPath := filepath.Join("reports", "verification.json")

		if _, err := os.Stat(verPath); os.IsNotExist(err) {
			fmt.Println("Error: No verification report found. Please run 'go run . verify' first.")
			os.Exit(1)
		}

		var report verificationReport
		if err := utils.ReadJSON(verPath, &report); err != nil {
			return fmt.Errorf("failed to read verification report file: %w", err)
		}

		// Filter resources that still exist or failed verification
		var failedResources []struct {
			Service      string `json:"service"`
			ID           string `json:"id"`
			Name         string `json:"name"`
			Verification string `json:"verification"`
			Reason       string `json:"reason"`
		}

		for _, r := range report.Resources {
			if r.Verification == "EXISTS" || r.Verification == "FAILED" {
				failedResources = append(failedResources, r)
			}
		}

		if len(failedResources) == 0 {
			fmt.Println("No active resources remain in AWS. All planned resources are verified as successfully deleted!")
			return nil
		}

		fmt.Println("AWS Kill Switch - Troubleshooting Guide")
		fmt.Println("────────────────────────────────────────────")

		for _, r := range failedResources {
			why := "AWS API indicates the resource still remains active or failed verification check."
			fix := "Check the AWS console for active dependencies or retry execution using 'go run . kill'."

			serviceLower := strings.ToLower(r.Service)

			if strings.Contains(serviceLower, "s3 bucket") || strings.Contains(serviceLower, "s3") {
				why = "The S3 bucket still exists because it contains files, active object versions, or delete markers."
				fix = fmt.Sprintf("Run: aws s3 rb s3://%s --force\n            Or empty the bucket contents in the AWS Console.", r.ID)
			} else if strings.Contains(serviceLower, "security group") {
				why = "This security group still exists because it is cross-referenced by another security group's ingress/egress rule, or attached to an active Network Interface (ENI)."
				fix = "Revoke ingress/egress rules that cross-reference this group, or wait for dependent ENIs to detach."
			} else if strings.Contains(serviceLower, "cloudfront distribution") || strings.Contains(serviceLower, "cloudfront") {
				why = "CloudFront distributions must be disabled and fully deployed globally before they can be deleted."
				fix = fmt.Sprintf("Disable the distribution %s in the AWS Console, wait for its status to be 'Deployed', and then delete.", r.ID)
			} else if r.Reason != "" {
				why = fmt.Sprintf("Verification failed: %s", r.Reason)
			}

			fmt.Printf("[%s] %s\n", r.Service, r.Name)
			fmt.Printf("  Status  : Failed\n")
			fmt.Printf("  Why     : %s\n", why)
			fmt.Printf("  Fix     : %s\n", fix)
			fmt.Println("────────────────────────────────────────────")
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(explainCmd)
}
