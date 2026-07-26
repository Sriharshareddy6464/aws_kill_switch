package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/engine"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/services"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var (
	force  bool
	dryRun bool
)

type terminalReporter struct{}

func (tr *terminalReporter) OnStart(index int, total int, res models.Resource) {
	cleanType := cleanKillTypeName(res.Type)
	name := res.Name
	if name == "" {
		name = res.ID
	}
	fmt.Printf("[%d/%d] %s : %s...... Status  : Deleting...", index, total, cleanType, name)
}

func (tr *terminalReporter) OnSuccess(index int, total int, res models.Resource) {
	cleanType := cleanKillTypeName(res.Type)
	name := res.Name
	if name == "" {
		name = res.ID
	}
	// Clear active Deleting line
	fmt.Print("\r\033[K")
	// Print green tick success line
	fmt.Printf("\033[32m✓ %s : %s has successfully deleted\033[0m\n", cleanType, name)
	// Delay briefly so user sees the tick, then clear/hide it
	time.Sleep(400 * time.Millisecond)
	fmt.Print("\033[1A\033[K") // move up one line and clear it
}

func (tr *terminalReporter) OnFailure(index int, total int, res models.Resource, err error) {
	cleanType := cleanKillTypeName(res.Type)
	name := res.Name
	if name == "" {
		name = res.ID
	}
	// Clear active Deleting line
	fmt.Print("\r\033[K")

	// Check if this is a CloudFront lifecycle error
	if lifeErr, ok := err.(*services.CloudFrontLifecycleError); ok {
		fmt.Printf("\033[31m✗ %s : %s deletion failed.\033[0m\n", cleanType, name)
		fmt.Println()
		fmt.Println("CloudFront Distribution")
		fmt.Println()
		fmt.Printf("Status : %s\n", lifeErr.Status)
		fmt.Println()
		fmt.Println("AWS Error")
		fmt.Println(lifeErr.AWSError)
		fmt.Println()
		fmt.Println("Reason")
		fmt.Println(lifeErr.Reason)
		fmt.Println()
		fmt.Println("Next Action")
		fmt.Println(lifeErr.NextAction)
		fmt.Println("────────────────────────────────────────────")
		return
	}

	// Print red cross failure line
	fmt.Printf("\033[31m✗ %s : %s deletion failed.\033[0m\n", cleanType, name)
	fmt.Printf("\033[31mReason : %v\033[0m\n", err)
	fmt.Println("────────────────────────────────────────────")
}

func (tr *terminalReporter) OnPending(index int, total int, res models.Resource, err error) {
	cleanType := cleanKillTypeName(res.Type)
	name := res.Name
	if name == "" {
		name = res.ID
	}
	// Clear active Deleting line
	fmt.Print("\r\033[K")

	// Print yellow warning pending line
	fmt.Printf("\033[33m⚠ %s : %s has pending propagation.\033[0m\n", cleanType, name)
	if pendingErr, ok := err.(*services.CloudFrontPendingError); ok {
		fmt.Println()
		fmt.Println("CloudFront Distribution")
		fmt.Println()
		fmt.Println("Status : Pending")
		fmt.Println()
		fmt.Println("AWS Error")
		fmt.Println(pendingErr.AWSError)
		fmt.Println()
		fmt.Println("Reason")
		fmt.Println(pendingErr.Reason)
		fmt.Println()
		fmt.Println("Next Action")
		fmt.Println(pendingErr.NextAction)
	} else {
		fmt.Printf("Reason : %v\n", err)
	}
	fmt.Println("────────────────────────────────────────────")
}

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Execute planned resource deletions in order",
	Long:  `Sequentially destroys the infrastructure listed in the plan, checking for dependency releases and polling AWS. Requires a completed plan first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		planPath := filepath.Join("reports", "plan.json")

		// Guard check: plan must exist
		if _, err := os.Stat(planPath); os.IsNotExist(err) {
			fmt.Printf("Error: No execution plan found at %s. Please run 'aws-kill plan' first.\n", planPath)
			os.Exit(1)
		}

		fmt.Println("AWS Kill Switch - Execution")
		fmt.Println("────────────────────────────────────────────")
		fmt.Print("plan loading [                      ] 0%")

		// Load execution plan
		var plan models.Plan
		if err := utils.ReadJSON(planPath, &plan); err != nil {
			fmt.Println()
			return fmt.Errorf("failed to read execution plan: %w", err)
		}

		// Animate plan loading progress bar (0% -> 100%)
		dots := 20
		for i := 0; i <= 100; i += 5 {
			filled := (i * dots) / 100
			bar := ""
			for j := 0; j < dots; j++ {
				if j < filled {
					bar += ":"
				} else {
					bar += " "
				}
			}
			fmt.Printf("\rplan loading [ %s ] %d%%", bar, i)
			time.Sleep(30 * time.Millisecond)
		}
		// Clear loading line
		fmt.Print("\r\033[K")
		// Print green tick success line
		fmt.Println("✓ Plan loaded successfully. [:::::::::::::::::::::::::::::100%]")
		fmt.Printf("Resources Scheduled : %d\n", len(plan.Steps))
		fmt.Println("────────────────────────────────────────────")

		if len(plan.Steps) == 0 {
			fmt.Println("Plan is empty. Nothing to destroy.")
			return nil
		}

		// Redirect utils.Logger to local log file to clean output logs
		rawLogFile, err := os.OpenFile(filepath.Join("reports", "raw_execution.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err == nil {
			defer rawLogFile.Close()
			utils.Logger = slog.New(slog.NewTextHandler(rawLogFile, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
		} else {
			utils.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
		}

		// Reset further downstream execution status
		os.Remove(filepath.Join("reports", "verification.json"))

		// Initialize AWS Session configuration
		prof := viper.GetString("profile")
		reg := viper.GetString("region")
		awsCfg, err := aws.NewSession(cmd.Context(), aws.Config{
			Profile: prof,
			Region:  reg,
		})
		if err != nil {
			return fmt.Errorf("failed to initialize AWS config: %w", err)
		}

		// Run Killer Engine
		killer := engine.NewKiller(awsCfg, dryRun)
		killer.Reporter = &terminalReporter{}

		result, err := killer.Kill(cmd.Context(), &plan)
		if err != nil {
			return fmt.Errorf("execution of deletion plan failed: %w", err)
		}

		// Ensure reports folder exists
		if err := os.MkdirAll("reports", 0755); err != nil {
			return fmt.Errorf("failed to create reports directory: %w", err)
		}

		// Write result file
		resultPath := filepath.Join("reports", "result.json")
		if err := utils.WriteJSON(resultPath, result); err != nil {
			return fmt.Errorf("failed to write result JSON: %w", err)
		}

		// Completion message
		fmt.Printf("[%d/%d] All planned resources have been processed.\n", len(plan.Steps), len(plan.Steps))
		fmt.Println("────────────────────────────────────────────")

		statusText := "SUCCESS"
		if len(result.FailedResources) > 0 {
			if len(result.DeletedResources) > 0 || len(result.PendingResources) > 0 {
				statusText = "PARTIAL SUCCESS"
			} else {
				statusText = "FAILED"
			}
		} else if len(result.PendingResources) > 0 {
			if len(result.DeletedResources) > 0 {
				statusText = "PARTIAL SUCCESS"
			} else {
				statusText = "PENDING"
			}
		}

		if statusText == "SUCCESS" {
			fmt.Println("\033[1;32mALL PLANNED RESOURCES HAVE BEEN TERMINATED SUCCESSFULLY\033[0m")
			fmt.Println()
		}

		fmt.Println("Execution Summary")
		fmt.Printf("Resources Scheduled    : %d\n", len(plan.Steps))
		fmt.Printf("Successfully Deleted   : %d\n", len(result.DeletedResources))
		if len(result.PendingResources) > 0 {
			fmt.Printf("Pending                : %d\n", len(result.PendingResources))
		}
		fmt.Printf("Failed                 : %d\n", len(result.FailedResources))
		fmt.Printf("Execution Status       : %s\n", statusText)
		fmt.Printf("Result Report          : %s\n", resultPath)
		fmt.Println("for verification  run \"go run . verify \"")

		return nil
	},
}

func cleanKillTypeName(t string) string {
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
	killCmd.Flags().BoolVar(&force, "force", false, "Bypass interactive confirmation prompt")
	killCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate deletion without executing real AWS changes")
	RootCmd.AddCommand(killCmd)
}
