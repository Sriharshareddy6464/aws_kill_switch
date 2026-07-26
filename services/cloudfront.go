package services

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/Sriharshareddy6464/aws-kill/models"
)

type CloudFrontLifecycleError struct {
	Status     string
	AWSError   string
	Reason     string
	NextAction string
	RawError   error
}

func (e *CloudFrontLifecycleError) Error() string {
	return e.RawError.Error()
}

type CloudFrontPendingError struct {
	AWSError   string
	Reason     string
	NextAction string
	RawError   error
}

func (e *CloudFrontPendingError) Error() string {
	return e.Reason
}

type CloudFrontService struct {
	Client *cloudfront.Client
}

func NewCloudFrontService(client *cloudfront.Client) *CloudFrontService {
	return &CloudFrontService{Client: client}
}

func (s *CloudFrontService) Scan(ctx context.Context, tagFilter string) ([]models.Resource, map[string]int, error) {
	var resources []models.Resource
	counts := map[string]int{"Distributions": 0}
	input := &cloudfront.ListDistributionsInput{}
	result, err := s.Client.ListDistributions(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	if result.DistributionList == nil {
		return resources, counts, nil
	}

	for _, dist := range result.DistributionList.Items {
		counts["Distributions"]++
		resources = append(resources, models.Resource{
			ID:     *dist.Id,
			Name:   *dist.DomainName,
			Type:   "CloudFront",
			Region: "",
		})
	}
	return resources, counts, nil
}

func (s *CloudFrontService) Delete(ctx context.Context, id string) error {
	// Step 1: Fetch distribution info to get current ETag
	dist, err := s.Client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: &id,
	})
	if err != nil {
		if isNoSuchDistribution(err) {
			return nil
		}
		return err
	}

	// Try direct deletion using the retrieved ETag
	_, deleteErr := s.Client.DeleteDistribution(ctx, &cloudfront.DeleteDistributionInput{
		Id:      &id,
		IfMatch: dist.ETag,
	})
	if deleteErr == nil {
		return nil // Success!
	}

	// Deletion failed. Evaluate the state to determine next action.
	errLower := strings.ToLower(deleteErr.Error())

	// Case 2: Distribution is Enabled (or error says it's not disabled)
	if strings.Contains(errLower, "distributionnotdisabled") || (dist.Distribution != nil && dist.Distribution.DistributionConfig != nil && *dist.Distribution.DistributionConfig.Enabled) {
		// Fetch config and ETag to disable it
		configResult, getConfErr := s.Client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
			Id: &id,
		})
		if getConfErr != nil {
			return &CloudFrontLifecycleError{
				Status:     "Failed",
				AWSError:   "GetConfigFailed",
				Reason:     "Failed to retrieve distribution configuration to disable it.",
				NextAction: "Check your AWS credentials/permissions or try again later.",
				RawError:   getConfErr,
			}
		}

		config := configResult.DistributionConfig
		config.Enabled = aws.Bool(false)

		_, updateErr := s.Client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
			Id:                 &id,
			DistributionConfig: config,
			IfMatch:            configResult.ETag,
		})
		if updateErr != nil {
			return &CloudFrontLifecycleError{
				Status:     "Failed",
				AWSError:   extractErrorCode(updateErr),
				Reason:     "Failed to disable the distribution.",
				NextAction: "Check the raw AWS error and ensure you have permissions to UpdateDistribution.",
				RawError:   updateErr,
			}
		}

		// Disabling update was successful. Transition to Pending propagation status.
		return &CloudFrontPendingError{
			AWSError:   "DistributionNotDisabled",
			Reason:     "CloudFront distributions must be disabled before they can be deleted. The distribution has been successfully disabled and is now propagating.",
			NextAction: "Wait for the distribution status to become Deployed and run the Kill command again.",
			RawError:   deleteErr,
		}
	}

	// Case 3: Distribution is Already Disabled (but deletion failed due to other reasons like global deployment in progress)
	why := "CloudFront distribution deletion is currently blocked."
	nextAction := "Wait for the distribution status to become Deployed and run the Kill command again."
	awsError := extractErrorCode(deleteErr)

	if strings.Contains(errLower, "invalidifmatchversion") {
		why = "The If-Match version is missing or not valid for the resource. Global deployment of the disabled distribution might still be in progress."
		nextAction = "Wait for the global deployment to finish (Status: Deployed) and run the Kill command again."
	} else if strings.Contains(errLower, "invalidstate") {
		why = "The distribution is in an invalid state for deletion (e.g. still deploying or changing states)."
		nextAction = "Wait for the status to become Deployed and retry."
	}

	return &CloudFrontLifecycleError{
		Status:     "Failed",
		AWSError:   awsError,
		Reason:     why,
		NextAction: nextAction,
		RawError:   deleteErr,
	}
}

func extractErrorCode(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	for _, code := range []string{"DistributionNotDisabled", "InvalidIfMatchVersion", "NoSuchDistribution", "InvalidState", "AccessDenied", "PreconditionFailed"} {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(code)) {
			return code
		}
	}
	return "UnknownError"
}

func isNoSuchDistribution(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "nosuchdistribution") || strings.Contains(errStr, "404")
}
