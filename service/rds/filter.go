package rds

import (
	"github.com/aws/aws-sdk-go/aws"
	observer "github.com/imkira/go-observer"
	"github.com/seatgeek/aws-dynamic-consul-catalog/config"
	log "github.com/sirupsen/logrus"
)

func (r *RDS) filter(all, filtered observer.Property) {
	logger := log.WithField("worker", "filter")
	logger.Info("Starting RDS instance filter worker")
	stream := all.Observe()

	for {
		select {
		case <-r.quitCh:
			return

		// wait for changes
		case <-stream.Changes():
			logger.Debug("Starting filtering RDS instances")

			stream.Next()
			instances := stream.Value().([]*config.DBInstance)

			filteredInstances := make([]*config.DBInstance, 0)

			for _, instance := range instances {
				if !r.filterByInstanceData(instance, r.instanceFilters) {
					continue
				}

				if !r.filterByInstanceTags(instance, r.tagFilters) {
					continue
				}

				filteredInstances = append(filteredInstances, instance)
			}

			filtered.Update(filteredInstances)
			logger.Debug("Finished filtering RDS instances")
		}
	}
}

func (r *RDS) filterByInstanceData(instance *config.DBInstance, filters config.Filters) bool {
	if len(filters) == 0 {
		return true
	}

	for k, v := range filters {
		switch k {
		case "AvailabilityZone":
			return v == aws.StringValue(instance.AvailabilityZone)
		case "DBInstanceArn":
			return v == aws.StringValue(instance.DBInstanceArn)
		case "DBInstanceClass":
			return v == aws.StringValue(instance.DBInstanceClass)
		case "DBInstanceIdentifier":
			return v == aws.StringValue(instance.DBInstanceIdentifier)
		case "DBInstanceStatus":
			return v == aws.StringValue(instance.DBInstanceStatus)
		case "Engine":
			return v == aws.StringValue(instance.Engine)
		case "EngineVersion":
			return v == aws.StringValue(instance.EngineVersion)
		case "VpcId":
			return v == aws.StringValue(instance.DBSubnetGroup.VpcId)
		default:
			log.Fatalf("Unknown instance filter key %s (%s)", k, v)
		}
	}

	return true
}

func (r *RDS) filterByInstanceTags(instance *config.DBInstance, filters config.Filters) bool {
	if len(filters) == 0 {
		return true
	}

	tags := instance.Tags

	for k, v := range filters {
		val, ok := tags[k]

		// the tag key doesn't exist
		if !ok {
			return false
		}

		// the value doesn't match
		if val != v {
			return false
		}
	}

	return true
}
