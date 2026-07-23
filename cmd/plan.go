package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/Sriharshareddy6464/aws-kill/engine"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate deletion plan based on resource dependencies",
	Long:  `Analyzes relationships between scanned resources and maps out an optimal, safe deletion sequence. Includes interactive TUI selection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inventoryPath := filepath.Join("reports", "inventory.json")

		// Guard check: inventory must exist
		if _, err := os.Stat(inventoryPath); os.IsNotExist(err) {
			fmt.Printf("Error: No scan inventory found at %s. Please run 'aws-kill scan' first.\n", inventoryPath)
			os.Exit(1)
		}

		fmt.Println("Reading scan inventory...")
		var inventory models.Inventory
		if err := utils.ReadJSON(inventoryPath, &inventory); err != nil {
			return fmt.Errorf("failed to read inventory file: %w", err)
		}

		if len(inventory.Resources) == 0 {
			fmt.Println("Inventory is empty. Nothing to plan.")
			return nil
		}

		// Build a lookup map for transitive VPC resolution
		inventoryMap := make(map[string]models.Resource)
		for _, res := range inventory.Resources {
			inventoryMap[res.ID] = res
		}

		// Group resources by VPC or Global/Independent categories
		groups := make(map[string][]models.Resource)
		for _, res := range inventory.Resources {
			vpcID := resolveVPC(res, inventoryMap)
			var groupName string
			if vpcID != "" {
				groupName = fmt.Sprintf("VPC: %s", vpcID)
			} else {
				groupName = "Global / Independent Resources"
			}
			groups[groupName] = append(groups[groupName], res)
		}

		// Prepare list of options for the TUI prompt
		var groupNames []string
		for name := range groups {
			groupNames = append(groupNames, name)
		}
		// Sort group names so "Global / Independent Resources" is always last, VPCs first alphabetically
		sort.Slice(groupNames, func(i, j int) bool {
			if groupNames[i] == "Global / Independent Resources" {
				return false
			}
			if groupNames[j] == "Global / Independent Resources" {
				return true
			}
			return groupNames[i] < groupNames[j]
		})

		// Render group options showing counts and types
		var options []string
		for _, name := range groupNames {
			resList := groups[name]
			// Summarize types in this group
			typeCounts := make(map[string]int)
			for _, r := range resList {
				typeCounts[r.Type]++
			}
			var typeSummary []string
			for t, c := range typeCounts {
				typeSummary = append(typeSummary, fmt.Sprintf("%s (%d)", t, c))
			}
			options = append(options, fmt.Sprintf("%-35s [%d resources: %s]", name, len(resList), strings.Join(typeSummary, ", ")))
		}

		// Interactive TUI Selection
		var selectedOptions []string
		prompt := &survey.MultiSelect{
			Message:  "Select resource groups to target for deletion:",
			Options:  options,
			Default:  options, // Select all by default
			PageSize: 10,
		}
		if err := survey.AskOne(prompt, &selectedOptions); err != nil {
			return fmt.Errorf("failed during interactive group selection: %w", err)
		}

		if len(selectedOptions) == 0 {
			fmt.Println("No groups selected. Planning aborted.")
			return nil
		}

		// Map selected option strings back to resource slices
		var filteredResources []models.Resource
		for _, opt := range selectedOptions {
			// Find the matching groupName from the option prefix
			for _, name := range groupNames {
				if strings.HasPrefix(opt, name) {
					filteredResources = append(filteredResources, groups[name]...)
					break
				}
			}
		}

		fmt.Printf("\nSelected %d resources for planning.\n", len(filteredResources))

		// Clean up further downstream files
		os.Remove(filepath.Join("reports", "result.json"))
		os.Remove(filepath.Join("reports", "verification.json"))

		// Run Planner on the filtered subset
		planner := engine.NewPlanner()
		plan, err := planner.Plan(cmd.Context(), &models.Inventory{Resources: filteredResources})
		if err != nil {
			return fmt.Errorf("planning failed: %w", err)
		}

		// Output plan details as a structured list
		fmt.Println("\nGenerated AWS Deletion Order:")
		fmt.Println("------------------------------------------------")
		for i, step := range plan.Steps {
			name := step.Name
			if name == "" {
				name = "<no-name>"
			}
			fmt.Printf("%3d. %-30s [%-25s] ID: %s\n", i+1, name, step.Type, step.ID)
		}
		fmt.Println("------------------------------------------------")

		// Final Planning Stage Confirmation
		var confirm bool
		confirmPrompt := &survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to write this deletion plan containing %d steps? Once written, running 'kill' will destroy them without further prompts.", len(plan.Steps)),
			Default: false,
		}
		if err := survey.AskOne(confirmPrompt, &confirm); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		if !confirm {
			fmt.Println("Planning aborted. Plan file was not generated.")
			return nil
		}

		// Ensure reports folder exists
		if err := os.MkdirAll("reports", 0755); err != nil {
			return fmt.Errorf("failed to create reports directory: %w", err)
		}

		// Write plan to file
		planPath := filepath.Join("reports", "plan.json")
		if err := utils.WriteJSON(planPath, plan); err != nil {
			return fmt.Errorf("failed to write plan JSON: %w", err)
		}

		fmt.Printf("Plan created successfully. %d deletion steps saved to %s.\n", len(plan.Steps), planPath)
		return nil
	},
}

// resolveVPC searches for a VPC ID in the resource or its dependencies recursively
func resolveVPC(res models.Resource, inventoryMap map[string]models.Resource) string {
	if strings.HasPrefix(res.ID, "vpc-") {
		return res.ID
	}

	for _, dep := range res.Dependencies {
		if strings.HasPrefix(dep, "vpc-") {
			return dep
		}
	}

	// Transitive search
	for _, dep := range res.Dependencies {
		if depRes, ok := inventoryMap[dep]; ok {
			if vpcID := resolveVPC(depRes, inventoryMap); vpcID != "" {
				return vpcID
			}
		}
	}

	return ""
}

func init() {
	RootCmd.AddCommand(planCmd)
}
