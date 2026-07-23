package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/engine"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var (
	force  bool
	dryRun bool
)

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

		fmt.Println("Loading execution plan...")
		var plan models.Plan
		if err := utils.ReadJSON(planPath, &plan); err != nil {
			return fmt.Errorf("failed to read execution plan: %w", err)
		}

		if len(plan.Steps) == 0 {
			fmt.Println("Plan is empty. Nothing to destroy.")
			return nil
		}

		fmt.Println("Executing deletion plan...")

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

		fmt.Println("------------------------------------------------")
		fmt.Printf("Execution completed.\nDeleted resources: %d\nFailed resources:  %d\nResults saved to %s.\n",
			len(result.DeletedResources), len(result.FailedResources), resultPath)

		return nil
	},
}

func init() {
	killCmd.Flags().BoolVar(&force, "force", false, "Bypass interactive confirmation prompt")
	killCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate deletion without executing real AWS changes")
	RootCmd.AddCommand(killCmd)
}
