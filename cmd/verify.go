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

type verificationOutcome struct {
	successLines            []string
	failureLines            []string
	reportResources         []map[string]interface{}
	verifiedDeletedCount    int
	stillExistingCount      int
	verificationErrorsCount int
	err                     error
}

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

		// Pre-load expected plan in main thread
		var plan models.Plan
		if err := utils.ReadJSON(planPath, &plan); err != nil {
			return fmt.Errorf("failed to read plan: %w", err)
		}

		// Initialize background AWS check channel
		outcomeChan := make(chan verificationOutcome, 1)

		// Start live AWS query thread in the background
		go func() {
			prof := viper.GetString("profile")
			reg := viper.GetString("region")
			awsCfg, err := aws.NewSession(cmd.Context(), aws.Config{
				Profile: prof,
				Region:  reg,
			})
			if err != nil {
				outcomeChan <- verificationOutcome{err: err}
				return
			}

			verifier := engine.NewVerifier(awsCfg)

			var successLines []string
			var failureLines []string
			var reportResources []map[string]interface{}

			verifiedDeletedCount := 0
			stillExistingCount := 0
			verificationErrorsCount := 0

			for _, res := range plan.Steps {
				cleanType := cleanVerifyTypeName(res.Type)
				name := res.Name
				if name == "" {
					name = res.ID
				}

				state, checkErr := verifier.VerifyResource(cmd.Context(), res)

				var repStatus string
				if state == "DELETED" {
					successLines = append(successLines, fmt.Sprintf("\033[32m✓\033[0m %-28s : %s", cleanType, name))
					repStatus = "DELETED"
					verifiedDeletedCount++
				} else if state == "EXISTS" {
					failureLines = append(failureLines, fmt.Sprintf("\033[31m✗\033[0m %-28s : %s (\033[31mStill Exists\033[0m)", cleanType, name))
					repStatus = "EXISTS"
					stillExistingCount++
				} else {
					failureLines = append(failureLines, fmt.Sprintf("\033[31m✗\033[0m %-28s : %s (\033[31mVerification Failed: %v\033[0m)", cleanType, name, checkErr))
					repStatus = "FAILED"
					verificationErrorsCount++
				}

				reportResources = append(reportResources, map[string]interface{}{
					"service":      cleanType,
					"id":           res.ID,
					"name":         name,
					"verification": repStatus,
				})
			}

			outcomeChan <- verificationOutcome{
				successLines:            successLines,
				failureLines:            failureLines,
				reportResources:         reportResources,
				verifiedDeletedCount:    verifiedDeletedCount,
				stillExistingCount:      stillExistingCount,
				verificationErrorsCount: verificationErrorsCount,
			}
		}()

		// --- TUI Loading Sequence (LED Wave Dots) ---
		fmt.Print("retrieving")
		for i := 0; i < 6; i++ {
			time.Sleep(500 * time.Millisecond)
			fmt.Print(".")
		}
		time.Sleep(300 * time.Millisecond)
		fmt.Print("\r\033[K") // Clear line

		fmt.Print("processing")
		for i := 0; i < 6; i++ {
			time.Sleep(500 * time.Millisecond)
			fmt.Print(".")
		}
		time.Sleep(300 * time.Millisecond)
		fmt.Print("\r\033[K") // Clear line

		fmt.Print("organising")
		for i := 0; i < 5; i++ {
			time.Sleep(500 * time.Millisecond)
			fmt.Print(".")
		}
		time.Sleep(400 * time.Millisecond)
		fmt.Print("\r\033[K") // Clear line

		// Fetch background result
		outcome := <-outcomeChan
		if outcome.err != nil {
			return fmt.Errorf("verification query failed: %w", outcome.err)
		}

		// --- Present Verification Report Line-by-Line ---
		printLineSlow("AWS Kill Switch - Verification")
		printLineSlow("")
		printLineSlow(fmt.Sprintf("Resources Expected To Be Deleted : %d", len(plan.Steps)))
		printLineSlow("────────────────────────────────────────────")
		printLineSlow("\033[1;32mSuccessfully Deleted\033[0m")
		if len(outcome.successLines) > 0 {
			for _, line := range outcome.successLines {
				printLineSlow(line)
			}
		} else {
			printLineSlow("  No successfully deleted resources found.")
		}
		printLineSlow("────────────────────────────────────────────")
		printLineSlow("\033[1;31mFailed Termination\033[0m")
		if len(outcome.failureLines) > 0 {
			for _, line := range outcome.failureLines {
				printLineSlow(line)
			}
		} else {
			printLineSlow("  No failed resources remaining.")
		}
		printLineSlow("────────────────────────────────────────────")
		printLineSlow("")

		// Calculate Verification Status and format output text
		var status string
		var statusText string

		if outcome.stillExistingCount == 0 && outcome.verificationErrorsCount == 0 {
			status = "SUCCESS"
			statusText = "\033[1;32mALL TERMINATION SUCCESS\033[0m"
		} else if outcome.verifiedDeletedCount > 0 {
			status = "PARTIAL_SUCCESS"
			statusText = "\033[5;33mPARTIAL SUCCESS\033[0m"
		} else {
			status = "FAILED"
			statusText = "\033[1;91mFAILED TERMINATION\033[0m"
		}

		// Save verification.json report
		verificationPath := filepath.Join("reports", "verification.json")
		reportData := map[string]interface{}{
			"generated_at": time.Now().Format(time.RFC3339),
			"status":       status,
			"summary": map[string]interface{}{
				"planned":             len(plan.Steps),
				"verified_deleted":    outcome.verifiedDeletedCount,
				"still_existing":      outcome.stillExistingCount,
				"verification_errors": outcome.verificationErrorsCount,
			},
			"resources": outcome.reportResources,
		}

		if err := utils.WriteJSON(verificationPath, reportData); err != nil {
			return fmt.Errorf("failed to save verification report: %w", err)
		}

		// Display Summary Line-by-Line
		printLineSlow("Execution Summary")
		printLineSlow("")
		printLineSlow(fmt.Sprintf("Resources Planned          : %d", len(plan.Steps)))
		printLineSlow(fmt.Sprintf("Verified Deleted           : %d", outcome.verifiedDeletedCount))
		printLineSlow(fmt.Sprintf("Still Existing             : %d", outcome.stillExistingCount))
		printLineSlow(fmt.Sprintf("Verification Errors        : %d", outcome.verificationErrorsCount))
		printLineSlow(fmt.Sprintf("Verification Status        : %s", statusText))
		printLineSlow("")
		printLineSlow(fmt.Sprintf("Verification Report        : %s", verificationPath))

		return nil
	},
}

func printLineSlow(line string) {
	fmt.Println(line)
	time.Sleep(80 * time.Millisecond)
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
	RootCmd.AddCommand(verifyCmd)
}
