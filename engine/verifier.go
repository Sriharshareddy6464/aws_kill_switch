package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsRegistry "github.com/Sriharshareddy6464/aws-kill/aws"
	"github.com/Sriharshareddy6464/aws-kill/models"
)

type Verifier struct {
	Registry *awsRegistry.ClientRegistry
}

func NewVerifier(cfg aws.Config) *Verifier {
	return &Verifier{
		Registry: awsRegistry.NewClientRegistry(cfg),
	}
}

// VerifyResource queries live AWS to check if a specific resource is deleted, still exists, or failed
func (v *Verifier) VerifyResource(ctx context.Context, res models.Resource) (string, error) {
	var err error

	switch res.Type {
	case "EC2 Instances":
		var out *ec2.DescribeInstancesOutput
		out, err = v.Registry.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{res.ID},
		})
		if err == nil {
			if len(out.Reservations) == 0 || len(out.Reservations[0].Instances) == 0 {
				return "DELETED", nil
			}
			state := out.Reservations[0].Instances[0].State.Name
			if state == "terminated" {
				return "DELETED", nil
			}
			return "EXISTS", nil
		}

	case "Subnets":
		_, err = v.Registry.EC2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			SubnetIds: []string{res.ID},
		})

	case "Security Groups":
		_, err = v.Registry.EC2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{res.ID},
		})

	case "Route Tables":
		_, err = v.Registry.EC2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{res.ID},
		})

	case "Internet Gateway":
		_, err = v.Registry.EC2.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []string{res.ID},
		})

	case "NAT Gateway":
		var out *ec2.DescribeNatGatewaysOutput
		out, err = v.Registry.EC2.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{res.ID},
		})
		if err == nil {
			if len(out.NatGateways) == 0 || out.NatGateways[0].State == "deleted" {
				return "DELETED", nil
			}
			return "EXISTS", nil
		}

	case "Elastic IP":
		_, err = v.Registry.EC2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
			AllocationIds: []string{res.ID},
		})

	case "Application Load Balancer":
		_, err = v.Registry.ELB.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{
			LoadBalancerArns: []string{res.ID},
		})

	case "Target Groups":
		_, err = v.Registry.ELB.DescribeTargetGroups(ctx, &elasticloadbalancingv2.DescribeTargetGroupsInput{
			TargetGroupArns: []string{res.ID},
		})

	case "ECS":
		var out *ecs.DescribeClustersOutput
		out, err = v.Registry.ECS.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: []string{res.ID},
		})
		if err == nil {
			if len(out.Clusters) == 0 || out.Clusters[0].Status == nil || *out.Clusters[0].Status == "INACTIVE" {
				return "DELETED", nil
			}
			return "EXISTS", nil
		}

	case "ECR":
		_, err = v.Registry.ECR.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
			RepositoryNames: []string{res.ID},
		})

	case "RDS":
		_, err = v.Registry.RDS.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: &res.ID,
		})

	case "S3":
		_, err = v.Registry.S3.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: &res.ID,
		})

	case "CloudFront":
		_, err = v.Registry.CloudFront.GetDistribution(ctx, &cloudfront.GetDistributionInput{
			Id: &res.ID,
		})

	case "Volume":
		_, err = v.Registry.EC2.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
			VolumeIds: []string{res.ID},
		})

	case "Snapshot":
		_, err = v.Registry.EC2.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{
			SnapshotIds: []string{res.ID},
		})

	case "KeyPair":
		_, err = v.Registry.EC2.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
			KeyPairIds: []string{res.ID},
		})

	case "LaunchTemplate":
		_, err = v.Registry.EC2.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{
			LaunchTemplateIds: []string{res.ID},
		})

	case "NetworkInterface":
		_, err = v.Registry.EC2.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []string{res.ID},
		})

	default:
		return "EXISTS", fmt.Errorf("unknown resource type: %s", res.Type)
	}

	if err != nil {
		if isNotFoundError(err) {
			return "DELETED", nil
		}
		return "FAILED", err
	}

	return "EXISTS", nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "nosuch") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "incorrect state") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not exist")
}
