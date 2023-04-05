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

	errs := make([]string, 0)

	// Indicate activity
	am.StartAction("Retrieving EKS Node counts")

	// Create a new instance of the EKS service
	if allRegions {
		// Get the list of all enabled regions for this account
		regionsSlice := GetEC2Regions(sf.GetEC2InstanceService(""), am)

		// Loop through all of the regions
		for _, regionName := range regionsSlice {
			nodeCount += eksCountForSingleRegion(regionName, sf, am, &errs)
		}
	} else {
		// Get the EC2 counts for the region selected by this session
		nodeCount += eksCountForSingleRegion("", sf, am, &errs)
	}

	// Indicate end of activity
	am.EndAction("OK (%d)", color.Bold(nodeCount))

	// Print list of errors that happened while retrieving node counts
	for _, err := range errs {
		am.SubResourceError(err)
	}

	return nodeCount
}

func eksCountForSingleRegion(region string, sf ServiceFactory, am ActivityMonitor, errs *[]string) int {
	// Indicate activity
	am.Message(".")

	// Retrieve an EKS service
	eksSvc := sf.GetEKSService(region)

	// Construct our input to find all Clusters
	input := &eks.ListClustersInput{}

	nodeCount := 0
	err := eksSvc.ListClusters(input, func(clusterList *eks.ListClustersOutput, lastPage bool) bool {
		// Loop through each cluster list
		for _, cluster := range clusterList.Clusters {
			// Retrieve cluster info
			clusterInfo, err := eksSvc.DescribeCluster(&eks.DescribeClusterInput{
				Name: aws.String(*cluster),
			})

			// If an error is found, add error message to slice and move onto the next cluster
			if err != nil {
				*errs = append(*errs, fmt.Sprintf("Unable to retrieve cluster information for %s (%s)", *cluster, err))
				return true
			}

			// Create the Kubernetes API Client
			k8Svc := sf.GetK8Service(clusterInfo.Cluster)
			if k8Svc != nil {
				// Get list of nodes within the cluster
				nodes, err := k8Svc.ListNodes()

				// If an error is found, add error message to slice and move onto the next cluster
				if err != nil {
					*errs = append(*errs, fmt.Sprintf("Unable to retrieve nodes in cluster \"%s\" (%s)", *cluster, err))
					return true
				}

				nodeCount += len(nodes.Items)
			} else {
				*errs = append(*errs, "Unable to create a Kubernetes client")
			}
		}

		return true
	})

	if err != nil {
		*errs = append(*errs, fmt.Sprintf("Unable to list clusters for region %s (%s)", region, err))
	}

	return nodeCount
}