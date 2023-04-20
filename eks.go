/******************************************************************************
Cloud Resource Counter
File: eks.go

Summary: Provides a count of all EKS nodes.
******************************************************************************/

package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	color "github.com/logrusorgru/aurora"
)

// EKSNodes retrieves the count of all EKS Nodes either for all
// regions (allRegions is true) or the region associated with the
// session. This method gives status back to the user via the supplied
// ActivityMonitor instance.
func EKSNodes(sf ServiceFactory, am ActivityMonitor, allRegions bool) int {
	nodeCount := 0

	errs := make([]error, 0)

	// Indicate activity
	am.StartAction("Retrieving EKS Node counts")

	// Create a new instance of the EKS service
	regionsSlice := []string{""}
	if allRegions {
		regionsSlice = GetEC2Regions(sf.GetEC2InstanceService(""), am)
	}

	for _, regionName := range regionsSlice {
		count, eksErrs := eksCountForSingleRegion(regionName, sf, am)
		errs = append(errs, eksErrs...)
		nodeCount += count
	}

	// Indicate end of activity
	am.EndAction("OK (%d)", color.Bold(nodeCount))

	// Print list of errors that happened while retrieving node counts
	for _, err := range errs {
		am.SubResourceError(err.Error())
	}

	return nodeCount
}

func eksCountForSingleRegion(region string, sf ServiceFactory, am ActivityMonitor) (int, []error) {
	errs := make([]error, 0)

	// Indicate activity
	am.Message(".")

	// Retrieve an EKS service
	eksSvc := sf.GetEKSService(region)

	// Construct our input to find all Clusters
	clusterInput := &eks.ListClustersInput{}

	nodeCount := 0
	err := eksSvc.ListClusters(clusterInput, func(clusterList *eks.ListClustersOutput, _ bool) bool {
		// Loop through each cluster list
		for _, cluster := range clusterList.Clusters {
			count, err := countNodes(eksSvc, cluster)
			errs = append(errs, err...)
			nodeCount += count
		}
		return true
	})

	if err != nil {
		errs = append(errs, fmt.Errorf("unable to list clusters for region %s (%s)", region, err))
	}

	return nodeCount, errs
}

func countNodes(eksSvc *EKSService, cluster *string) (int, []error) {
	nodeCount := 0
	errs := make([]error, 0)
	nodeGroupsInput := &eks.ListNodegroupsInput{ClusterName: aws.String(*cluster)}

	err := eksSvc.ListNodeGroups(nodeGroupsInput, func(nodeGroupList *eks.ListNodegroupsOutput, _ bool) bool {
		// Loop through each nodegroup
		for _, nodeGroup := range nodeGroupList.Nodegroups {
			describeNodeGroupInput := &eks.DescribeNodegroupInput{
				ClusterName:   aws.String(*cluster),
				NodegroupName: aws.String(*nodeGroup),
			}

			// Retrieve nodegroup info
			nodeGroupInfo, err := eksSvc.DescribeNodegroups(describeNodeGroupInput)
			if err != nil {
				errs = append(errs, fmt.Errorf("unable to describe %s nodegroup (%s)", *nodeGroup, err))
				return true
			}

			// Add the node count for the nodepool
			nodeCount += int(*nodeGroupInfo.Nodegroup.ScalingConfig.DesiredSize)
		}
		return true
	})

	if err != nil {
		errs = append(errs, fmt.Errorf("unable to list nodegroups for %s cluster (%s)", *cluster, err))
	}

	return nodeCount, errs
}
