package engine

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/Sriharshareddy6464/aws-kill/models"
)

func TestKiller_DryRun(t *testing.T) {
	plan := &models.Plan{
		Steps: []models.Resource{
			{ID: "i-12345", Type: "EC2 Instances"},
			{ID: "subnet-abc", Type: "Subnets"},
		},
	}

	// In dry-run mode, we should not communicate with AWS APIs.
	killer := NewKiller(aws.Config{}, true)
	result, err := killer.Kill(context.Background(), plan)
	if err != nil {
		t.Fatalf("Killer failed: %v", err)
	}

	if len(result.DeletedResources) != 2 {
		t.Errorf("Expected 2 deleted resources, got %d", len(result.DeletedResources))
	}
	if len(result.FailedResources) != 0 {
		t.Errorf("Expected 0 failed resources, got %d", len(result.FailedResources))
	}

	if result.DeletedResources[0].ID != "i-12345" || result.DeletedResources[1].ID != "subnet-abc" {
		t.Errorf("Deleted resource mapping mismatch")
	}
}
