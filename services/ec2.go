package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/Sriharshareddy6464/aws-kill/models"
	"github.com/Sriharshareddy6464/aws-kill/utils"
)

type EC2Service struct {
	Client *ec2.Client
}

func NewEC2Service(client *ec2.Client) *EC2Service {
	return &EC2Service{Client: client}
}

func (s *EC2Service) Scan(ctx context.Context, tagFilter string) ([]models.Resource, map[string]int, error) {
	var resources []models.Resource
	counts := map[string]int{
		"Instances":             0,
		"Running Instances":     0,
		"Stopped Instances":     0,
		"Volumes":               0,
		"Snapshots":             0,
		"Key Pairs":             0,
		"Launch Templates":      0,
		"Placement Groups":      0,
		"Dedicated Hosts":       0,
		"Capacity Reservations": 0,
		"Network Interfaces":    0,
	}

	// 1. Describe Instances
	instResult, err := s.Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err == nil {
		for _, reservation := range instResult.Reservations {
			for _, inst := range reservation.Instances {
				if inst.State != nil && inst.State.Name == "terminated" {
					continue
				}

				counts["Instances"]++
				stateName := string(inst.State.Name)
				if stateName == "running" {
					counts["Running Instances"]++
				} else if stateName == "stopped" {
					counts["Stopped Instances"]++
				}

				tags := make(map[string]string)
				for _, t := range inst.Tags {
					tags[*t.Key] = *t.Value
				}

				resources = append(resources, models.Resource{
					ID:    *inst.InstanceId,
					Name:  tags["Name"],
					Type:  "EC2 Instances",
					State: stateName,
					Tags:  tags,
				})
			}
		}
	} else {
		utils.Logger.Warn("Skipped EC2 Instances describe", slog.Any("error", err))
	}

	// 2. Volumes
	if volResult, err := s.Client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{}); err == nil {
		counts["Volumes"] = len(volResult.Volumes)
		for _, vol := range volResult.Volumes {
			tags := make(map[string]string)
			for _, t := range vol.Tags {
				tags[*t.Key] = *t.Value
			}
			resources = append(resources, models.Resource{
				ID:    *vol.VolumeId,
				Name:  tags["Name"],
				Type:  "Volume",
				State: string(vol.State),
				Tags:  tags,
			})
		}
	} else {
		utils.Logger.Warn("Skipped EC2 Volumes describe", slog.Any("error", err))
	}

	// 3. Snapshots (OwnerId self to avoid millions of public ones)
	if snapResult, err := s.Client.DescribeSnapshots(ctx, &ec2.DescribeSnapshotsInput{OwnerIds: []string{"self"}}); err == nil {
		counts["Snapshots"] = len(snapResult.Snapshots)
		for _, snap := range snapResult.Snapshots {
			resources = append(resources, models.Resource{
				ID:   *snap.SnapshotId,
				Type: "Snapshot",
			})
		}
	} else {
		utils.Logger.Warn("Skipped EC2 Snapshots describe", slog.Any("error", err))
	}

	// 4. Key Pairs
	if kpResult, err := s.Client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{}); err == nil {
		counts["Key Pairs"] = len(kpResult.KeyPairs)
		for _, kp := range kpResult.KeyPairs {
			resources = append(resources, models.Resource{
				ID:   *kp.KeyPairId,
				Name: *kp.KeyName,
				Type: "KeyPair",
			})
		}
	} else {
		utils.Logger.Warn("Skipped EC2 Key Pairs describe", slog.Any("error", err))
	}

	// 5. Launch Templates
	if ltResult, err := s.Client.DescribeLaunchTemplates(ctx, &ec2.DescribeLaunchTemplatesInput{}); err == nil {
		counts["Launch Templates"] = len(ltResult.LaunchTemplates)
		for _, lt := range ltResult.LaunchTemplates {
			resources = append(resources, models.Resource{
				ID:   *lt.LaunchTemplateId,
				Name: *lt.LaunchTemplateName,
				Type: "LaunchTemplate",
			})
		}
	} else {
		utils.Logger.Warn("Skipped EC2 Launch Templates describe", slog.Any("error", err))
	}

	// 6. Placement Groups
	if pgResult, err := s.Client.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{}); err == nil {
		counts["Placement Groups"] = len(pgResult.PlacementGroups)
		for _, pg := range pgResult.PlacementGroups {
			resources = append(resources, models.Resource{
				ID:   *pg.GroupId,
				Name: *pg.GroupName,
				Type: "PlacementGroup",
			})
		}
	} else {
		utils.Logger.Warn("Skipped Placement Groups describe", slog.Any("error", err))
	}

	// 7. Dedicated Hosts
	if hostResult, err := s.Client.DescribeHosts(ctx, &ec2.DescribeHostsInput{}); err == nil {
		counts["Dedicated Hosts"] = len(hostResult.Hosts)
		for _, host := range hostResult.Hosts {
			resources = append(resources, models.Resource{
				ID:   *host.HostId,
				Type: "DedicatedHost",
			})
		}
	} else {
		utils.Logger.Warn("Skipped Dedicated Hosts describe", slog.Any("error", err))
	}

	// 8. Capacity Reservations
	if crResult, err := s.Client.DescribeCapacityReservations(ctx, &ec2.DescribeCapacityReservationsInput{}); err == nil {
		counts["Capacity Reservations"] = len(crResult.CapacityReservations)
		for _, cr := range crResult.CapacityReservations {
			resources = append(resources, models.Resource{
				ID:   *cr.CapacityReservationId,
				Type: "CapacityReservation",
			})
		}
	} else {
		utils.Logger.Warn("Skipped Capacity Reservations describe", slog.Any("error", err))
	}

	// 9. Network Interfaces (ENIs)
	if eniResult, err := s.Client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{}); err == nil {
		counts["Network Interfaces"] = len(eniResult.NetworkInterfaces)
		for _, eni := range eniResult.NetworkInterfaces {
			resources = append(resources, models.Resource{
				ID:   *eni.NetworkInterfaceId,
				Type: "NetworkInterface",
			})
		}
	} else {
		utils.Logger.Warn("Skipped Network Interfaces describe", slog.Any("error", err))
	}

	return resources, counts, nil
}

func (s *EC2Service) Delete(ctx context.Context, id string, resourceType string) error {
	switch resourceType {
	case "EC2 Instances":
		_, err := s.Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{id},
		})
		return err
	case "Volume":
		_, err := s.Client.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
			VolumeId: &id,
		})
		return err
	case "Snapshot":
		_, err := s.Client.DeleteSnapshot(ctx, &ec2.DeleteSnapshotInput{
			SnapshotId: &id,
		})
		return err
	case "KeyPair":
		_, err := s.Client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
			KeyPairId: &id,
		})
		return err
	case "LaunchTemplate":
		_, err := s.Client.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
			LaunchTemplateId: &id,
		})
		return err
	case "PlacementGroup":
		// Placement group uses GroupName
		_, err := s.Client.DeletePlacementGroup(ctx, &ec2.DeletePlacementGroupInput{
			GroupName: &id,
		})
		return err
	case "DedicatedHost":
		_, err := s.Client.ReleaseHosts(ctx, &ec2.ReleaseHostsInput{
			HostIds: []string{id},
		})
		return err
	case "CapacityReservation":
		_, err := s.Client.CancelCapacityReservation(ctx, &ec2.CancelCapacityReservationInput{
			CapacityReservationId: &id,
		})
		return err
	case "NetworkInterface":
		_, err := s.Client.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
			NetworkInterfaceId: &id,
		})
		return err
	default:
		return fmt.Errorf("unsupported EC2 resource type: %s", resourceType)
	}
}

