package services

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/Sriharshareddy6464/aws-kill/models"
)

type InternetGatewayService struct {
	Client *ec2.Client
}

func NewInternetGatewayService(client *ec2.Client) *InternetGatewayService {
	return &InternetGatewayService{Client: client}
}

func (s *InternetGatewayService) Scan(ctx context.Context, tagFilter string) ([]models.Resource, map[string]int, error) {
	var resources []models.Resource
	counts := map[string]int{"Internet Gateways": 0}
	input := &ec2.DescribeInternetGatewaysInput{}
	result, err := s.Client.DescribeInternetGateways(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	for _, igw := range result.InternetGateways {
		// Check if attached to a default VPC
		isDefaultAttached := false
		if len(igw.Attachments) > 0 {
			var vpcIds []string
			for _, att := range igw.Attachments {
				if att.VpcId != nil {
					vpcIds = append(vpcIds, *att.VpcId)
				}
			}
			if len(vpcIds) > 0 {
				vpcsOut, err := s.Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
					VpcIds: vpcIds,
				})
				if err == nil {
					for _, vpc := range vpcsOut.Vpcs {
						if vpc.IsDefault != nil && *vpc.IsDefault {
							isDefaultAttached = true
							break
						}
					}
				}
			}
		}

		if isDefaultAttached {
			continue
		}

		counts["Internet Gateways"]++
		var deps []string
		for _, att := range igw.Attachments {
			if att.VpcId != nil {
				deps = append(deps, *att.VpcId)
			}
		}

		tags := make(map[string]string)
		for _, t := range igw.Tags {
			tags[*t.Key] = *t.Value
		}

		resources = append(resources, models.Resource{
			ID:           *igw.InternetGatewayId,
			Name:         tags["Name"],
			Type:         "Internet Gateway",
			Region:       "",
			Dependencies: deps,
			Tags:         tags,
		})
	}
	return resources, counts, nil
}

func (s *InternetGatewayService) Delete(ctx context.Context, id string) error {
	input := &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{id},
	}
	res, err := s.Client.DescribeInternetGateways(ctx, input)
	if err == nil && len(res.InternetGateways) > 0 {
		for _, att := range res.InternetGateways[0].Attachments {
			if att.VpcId != nil {
				s.Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
					InternetGatewayId: &id,
					VpcId:             att.VpcId,
				})
			}
		}
	}

	_, err = s.Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: &id,
	})
	return err
}
