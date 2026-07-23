package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/engine"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Confirm target AWS resources are completely deleted",
	Long:  `Queries the AWS environment directly in real time to verify that all resources listed in plan.json are deleted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPath := filepath.Join("reports", "plan.json")

		// 1. Guard check: plan must exist to verify
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			fmt.Println("No execution plan found.")
			fmt.Println()
			fmt.Println("Run:")
			fmt.Println("go run . scan")
			fmt.Println("go run . plan")
			os.Exit(1)
		}

		fmt.Println("AWS Kill Switch - Verification")
		fmt.Print("Loading verification plan... [                    ] 0%")

		// Load expected plan
		var plan models.Plan
		if err := utils.ReadJSON(planPath, &plan); err != nil {
			fmt.Println()
			return fmt.Errorf("failed to read plan: %w", err)
		}

		// Animate verification plan loading bar
		dots := 20
		for i := 0; i <= 100; i += 10 {
			filled := (i * dots) / 100
			bar := ""
			for j := 0; j < dots; j++ {
				if j < filled {
					bar += "█"
				} else {
					bar += " "
				}
			}
			fmt.Printf("\rLoading verification plan... [%s] %d%%", bar, i)
			time.Sleep(20 * time.Millisecond)
		}
		fmt.Println()
		fmt.Println("✓ Plan loaded successfully.")
		fmt.Printf("Resources Expected To Be Deleted : %d\n", len(plan.Steps))
		fmt.Println("────────────────────────────────────────────")

		// Initialize AWS Session
		prof := viper.GetString("profile")
		reg := viper.GetString("region")
		awsCfg, err := aws.NewSession(cmd.Context(), aws.Config{
			Profile: prof,
			Region:  reg,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize AWS config: %w", err)
		}

		verifier := engine.NewVerifier(awsCfg)

		var reportResources []map[string]interface{}
		verifiedDeletedCount := 0
		stillExistingCount := 0
		verificationErrorsCount := 0

		// Verify resource-by-resource live
		for i, res := range plan.Steps {
			cleanType := cleanVerifyTypeName(res.Type)
			name := res.Name
			if name == "" {
				name = res.ID
			}

			fmt.Printf("[%d/%d]\n%s : %s\nStatus : Verifying...\n", i+1, len(plan.Steps), cleanType, name)

			state, checkErr := verifier.VerifyResource(cmd.Context(), res)

			// Clear "Status : Verifying..." line and check status
			fmt.Print("\033[1A\033[K") // Clear Verifying line

			var displayStatus string
			var repStatus string

			if state == "DELETED" {
				displayStatus = "\033[32m✓ Verified Deleted\033[0m"
				repStatus = "DELETED"
				verifiedDeletedCount++
			} else if state == "EXISTS" {
				displayStatus = "\033[31m✗ Resource Still Exists\033[0m"
				repStatus = "EXISTS"
				stillExistingCount++
			} else {
				displayStatus = fmt.Sprintf("\033[31m✗ Verification Failed: %v\033[0m", checkErr)
				repStatus = "FAILED"
				verificationErrorsCount++
			}

			fmt.Println(displayStatus)
			fmt.Println("────────────────────────────────────────────")

			reportResources = append(reportResources, map[string]interface{}{
				"service":      cleanType,
				"id":           res.ID,
				"name":         name,
				"verification": repStatus,
			})
		}

		// Calculate Verification Status
		status := "SUCCESS"
		if stillExistingCount > 0 || verificationErrorsCount > 0 {
			if verifiedDeletedCount > 0 {
				status = "PARTIAL_SUCCESS"
			} else {
				status = "FAILED"
			}
		}

		// Save verification.json report
		verificationPath := filepath.Join("reports", "verification.json")
		reportData := map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"status":       status,
			"summary": map[string]interface{}{
				"planned":             len(plan.Steps),
				"verified_deleted":    verifiedDeletedCount,
				"still_existing":      stillExistingCount,
				"verification_errors": verificationErrorsCount,
			},
			"resources": reportResources,
		}

		if err := utils.WriteJSON(verificationPath, reportData); err != nil {
			return fmt.Errorf("failed to save verification report: %w", err)
		}

		// Display Final Summary
		fmt.Println("Execution Summary")
		fmt.Println("────────────────────────────────────────────")
		fmt.Printf("Resources Planned          : %d\n", len(plan.Steps))
		fmt.Printf("Verified Deleted           : %d\n", verifiedDeletedCount)
		fmt.Printf("Still Existing             : %d\n", stillExistingCount)
		fmt.Printf("Verification Errors        : %d\n", verificationErrorsCount)
		fmt.Printf("Verification Status        : %s\n", status)
		fmt.Printf("Verification Report        : %s\n", verificationPath)

		if status == "FAILED" || stillExistingCount > 0 {
			os.Exit(1)
		}

		return nil
	},
}

func cleanVerifyTypeName(t string) string {
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
