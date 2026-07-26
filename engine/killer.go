package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	clientRegistry "github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/services"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

type ProgressReporter interface {
	OnStart(index int, total int, res models.Resource)
	OnSuccess(index int, total int, res models.Resource)
	OnFailure(index int, total int, res models.Resource, err error)
	OnPending(index int, total int, res models.Resource, err error)
}

type Killer struct {
	Registry *clientRegistry.ClientRegistry
	Config   aws.Config
	DryRun   bool
	Reporter ProgressReporter
}

func NewKiller(cfg aws.Config, dryRun bool) *Killer {
	return &Killer{
		Registry: clientRegistry.NewClientRegistry(cfg),
		Config:   cfg,
		DryRun:   dryRun,
	}
}

// Kill processes deletion steps sequentially
func (k *Killer) Kill(ctx context.Context, plan *models.Plan) (*models.Result, error) {
	result := &models.Result{
		DeletedResources: make([]models.Resource, 0),
		FailedResources:  make([]models.Resource, 0),
	}

	if plan == nil || len(plan.Steps) == 0 {
		return result, nil
	}

	// Initialize service instances
	ec2Svc := services.NewEC2Service(k.Registry.EC2)
	vpcSvc := services.NewVPCService(k.Registry.EC2)
	subnetSvc := services.NewSubnetService(k.Registry.EC2)
	sgSvc := services.NewSecurityGroupService(k.Registry.EC2)
	igwSvc := services.NewInternetGatewayService(k.Registry.EC2)
	ngwSvc := services.NewNATGatewayService(k.Registry.EC2)
	rtSvc := services.NewRouteTableService(k.Registry.EC2)
	eipSvc := services.NewElasticIPService(k.Registry.EC2)
	albSvc := services.NewALBService(k.Registry.ELB)
	tgSvc := services.NewTargetGroupService(k.Registry.ELB)
	ecsSvc := services.NewECSService(k.Registry.ECS)
	ecrSvc := services.NewECRService(k.Registry.ECR)
	rdsSvc := services.NewRDSService(k.Registry.RDS)
	s3Svc := services.NewS3Service(k.Registry.S3)
	cfSvc := services.NewCloudFrontService(k.Registry.CloudFront)

	for i, step := range plan.Steps {
		if k.Reporter != nil {
			k.Reporter.OnStart(i+1, len(plan.Steps), step)
		} else {
			utils.Logger.Info("Processing step", slog.Int("index", i+1), slog.String("type", step.Type), slog.String("id", step.ID))
		}

		if k.DryRun {
			// Simulate a short sleep to show animation progress in dry-run
			time.Sleep(300 * time.Millisecond)
			if k.Reporter != nil {
				k.Reporter.OnSuccess(i+1, len(plan.Steps), step)
			} else {
				utils.Logger.Info(fmt.Sprintf("[DRY-RUN] Successfully deleted %s (%s)", step.ID, step.Type))
			}
			result.DeletedResources = append(result.DeletedResources, step)
			continue
		}

		var err error
		switch step.Type {
		case "EC2 Instances", "Volume", "Snapshot", "KeyPair", "LaunchTemplate", "PlacementGroup", "DedicatedHost", "CapacityReservation", "NetworkInterface":
			err = ec2Svc.Delete(ctx, step.ID, step.Type)
		case "VPC":
			err = vpcSvc.Delete(ctx, step.ID)
		case "Subnets":
			err = subnetSvc.Delete(ctx, step.ID)
		case "Security Groups":
			err = sgSvc.Delete(ctx, step.ID)
		case "Internet Gateway":
			err = igwSvc.Delete(ctx, step.ID)
		case "NAT Gateway":
			err = ngwSvc.Delete(ctx, step.ID)
		case "Route Tables":
			err = rtSvc.Delete(ctx, step.ID)
		case "Elastic IP":
			err = eipSvc.Delete(ctx, step.ID)
		case "Application Load Balancer":
			err = albSvc.Delete(ctx, step.ID)
		case "Target Groups":
			err = tgSvc.Delete(ctx, step.ID)
		case "ECS":
			err = ecsSvc.Delete(ctx, step.ID)
		case "ECR":
			err = ecrSvc.Delete(ctx, step.ID)
		case "RDS":
			err = rdsSvc.Delete(ctx, step.ID)
		case "S3":
			err = s3Svc.Delete(ctx, step.ID)
		case "CloudFront":
			err = cfSvc.Delete(ctx, step.ID)
		default:
			err = fmt.Errorf("unknown resource type: %s", step.Type)
		}

		if err != nil {
			if pendingErr, ok := err.(*services.CloudFrontPendingError); ok {
				if k.Reporter != nil {
					k.Reporter.OnPending(i+1, len(plan.Steps), step, pendingErr)
				}
				step.State = "pending: " + pendingErr.Error()
				result.PendingResources = append(result.PendingResources, step)
				continue
			}

			if k.Reporter != nil {
				k.Reporter.OnFailure(i+1, len(plan.Steps), step, err)
			} else {
				utils.Logger.Error("Failed to delete resource", slog.String("id", step.ID), slog.Any("error", err))
			}
			step.State = "failed: " + err.Error()
			result.FailedResources = append(result.FailedResources, step)
			continue
		}

		// Wait/Poll for asynchronous resource termination
		k.waitForDeletion(ctx, step)

		if k.Reporter != nil {
			k.Reporter.OnSuccess(i+1, len(plan.Steps), step)
		} else {
			utils.Logger.Info("Successfully deleted resource", slog.String("id", step.ID))
		}
		step.State = "deleted"
		result.DeletedResources = append(result.DeletedResources, step)
	}

	return result, nil
}

// waitForDeletion handles polling loops for resources that are asynchronously deleted in AWS.
func (k *Killer) waitForDeletion(ctx context.Context, step models.Resource) {
	var checkFunc func() (bool, error)
	var interval, timeout time.Duration

	switch step.Type {
	case "EC2 Instances":
		interval = 5 * time.Second
		timeout = 5 * time.Minute
		checkFunc = func() (bool, error) {
			res, err := k.Registry.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				InstanceIds: []string{step.ID},
			})
			if err != nil {
				return true, nil // Missing or not found means terminated
			}
			if len(res.Reservations) == 0 || len(res.Reservations[0].Instances) == 0 {
				return true, nil
			}
			state := res.Reservations[0].Instances[0].State.Name
			return state == "terminated", nil
		}

	case "Volume":
		interval = 5 * time.Second
		timeout = 2 * time.Minute
		checkFunc = func() (bool, error) {
			_, err := k.Registry.EC2.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
				VolumeIds: []string{step.ID},
			})
			if err != nil {
				return true, nil // If error is NotFound, then it's deleted
			}
			return false, nil
		}

	case "NAT Gateway":
		interval = 10 * time.Second
		timeout = 10 * time.Minute
		checkFunc = func() (bool, error) {
			res, err := k.Registry.EC2.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
				NatGatewayIds: []string{step.ID},
			})
			if err != nil {
				return true, nil
			}
			if len(res.NatGateways) == 0 {
				return true, nil
			}
			state := res.NatGateways[0].State
			return state == "deleted", nil
		}

	case "Application Load Balancer":
		interval = 10 * time.Second
		timeout = 5 * time.Minute
		checkFunc = func() (bool, error) {
			_, err := k.Registry.ELB.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
				LoadBalancerArns: []string{step.ID},
			})
			if err != nil {
				return true, nil
			}
			return false, nil
		}

	case "RDS":
		interval = 15 * time.Second
		timeout = 15 * time.Minute
		checkFunc = func() (bool, error) {
			_, err := k.Registry.RDS.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: &step.ID,
			})
			if err != nil {
				return true, nil
			}
			return false, nil
		}
	}

	if checkFunc != nil {
		utils.Logger.Info("Waiting for resource deletion to complete...", slog.String("id", step.ID))
		err := clientRegistry.PollResource(ctx, checkFunc, interval, timeout)
		if err != nil {
			utils.Logger.Warn("Finished wait loop with non-critical warning", slog.String("id", step.ID), slog.Any("error", err))
		}
	}
}
