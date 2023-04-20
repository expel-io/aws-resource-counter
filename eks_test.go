/******************************************************************************
Cloud Resource Counter
File: s3_test.go

Summary: The Unit Test for s3.
******************************************************************************/

package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/expel-io/aws-resource-counter/mock"
)

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Fake EKS Clusters
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

//This simulates the minimal response from an AWS call
var fakeEKSClustersSlice = []*eks.ListClustersOutput{
	{
		Clusters: []*string{
			aws.String("cluster1"),
			aws.String("cluster2"),
			aws.String("cluster3"),
		},
	},
}
var size = int64(2)
var fakeEKSDescribeNodeGroup = &eks.DescribeNodegroupOutput{
	Nodegroup: &eks.Nodegroup{
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: &size,
		},
	},
}

var fakeEKSNodeGroupSlice = []*eks.ListNodegroupsOutput{
	{
		Nodegroups: []*string{
			aws.String("nodegroup-1"),
			aws.String("nodegroup-2"),
		},
	},
}

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Fake EKS Service
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

// To use this struct, the caller must supply a ListClustersOutput and DescribeClusterOutput
// struct. If it is missing, it will trigger the mock function to simulate an error from
// the corresponding function.
type fakeEKService struct {
	eksiface.EKSAPI
	LCResponse  []*eks.ListClustersOutput
	DNGResponse *eks.DescribeNodegroupOutput
	LNGResponse []*eks.ListNodegroupsOutput
}

func (feks *fakeEKService) DescribeNodegroup(input *eks.DescribeNodegroupInput) (*eks.DescribeNodegroupOutput, error) {
	// If there was no supplied response, then simulate a possible error
	if feks.DNGResponse == nil {
		return nil, errors.New("ListClusters returns an unexpected error: 2345")
	}

	return feks.DNGResponse, nil
}

// Simulate the ListClustersPages function
func (feks *fakeEKService) ListClustersPages(input *eks.ListClustersInput,
	fn func(*eks.ListClustersOutput, bool) bool) error {
	// If the supplied response is nil, then simulate an error
	if feks.LCResponse == nil {
		return errors.New("ListClustersPages encountered an unexpected error: 1234")
	}

	// Loop through the slice, invoking the supplied function
	for index, output := range feks.LCResponse {
		// Are we looking at the last "page" of our output?
		lastPage := index == len(feks.LCResponse)-1

		// Shall we exit our loop?
		if cont := fn(output, lastPage); !cont {
			break
		}
	}

	return nil
}

// Simulate the ListNodegroupsPages function
func (feks *fakeEKService) ListNodegroupsPages(input *eks.ListNodegroupsInput,
	fn func(*eks.ListNodegroupsOutput, bool) bool) error {
	// If the supplied response is nil, then simulate an error
	if feks.LNGResponse == nil {
		return errors.New("ListNodeGroups encountered an unexpected error: 1234")
	}

	// Loop through the slice, invoking the supplied function
	for index, output := range feks.LNGResponse {
		// Are we looking at the last "page" of our output?
		lastPage := index == len(feks.LNGResponse)-1

		// Shall we exit our loop?
		if cont := fn(output, lastPage); !cont {
			break
		}
	}

	return nil
}

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Fake Service Factory
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

// This structure simulates the AWS Service Factory by storing some pregenerated
// responses (that would come from AWS).
type fakeEKSServiceFactory struct {
	LCResponse  []*eks.ListClustersOutput
	DNGResponse *eks.DescribeNodegroupOutput
	LNGResponse []*eks.ListNodegroupsOutput
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) Init() {}

// Don't implement
func (fsf fakeEKSServiceFactory) GetCurrentRegion() string {
	return ""
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetAccountIDService() *AccountIDService {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetEC2InstanceService(string) *EC2InstanceService {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetRDSInstanceService(regionName string) *RDSInstanceService {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetS3Service() *S3Service {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetLambdaService(string) *LambdaService {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetContainerService(string) *ContainerService {
	return nil
}

// Don't need to implement
func (fsf fakeEKSServiceFactory) GetLightsailService(string) *LightsailService {
	return nil
}

func (fsf fakeEKSServiceFactory) GetEKSService(regionName string) *EKSService {
	return &EKSService{
		Client: &fakeEKService{
			LCResponse:  fsf.LCResponse,
			DNGResponse: fsf.DNGResponse,
			LNGResponse: fsf.LNGResponse,
		},
	}
}

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Unit Test for S3Buckets
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func TestEKSNodes(t *testing.T) {
	// Describe all of our test cases: 1 failure and 1 success
	cases := []struct {
		ExpectedCount                int
		ExpectErrorClusterList       bool
		ExpectErrorDescribeNodegroup bool
		ExpectErrorNodegroupList     bool
		name                         string
	}{
		// Expected count is 12 because there are 2 nodes defined for each node pool
		// each cluster has 2 node pools. so: 2 nodes * 2 node pools * 3 clusters
		{name: "the expected count is returned", ExpectedCount: 12},
		{name: "an error is logged for cluster list", ExpectErrorClusterList: true},
		{name: "an error is logged for describe nodegroup", ExpectErrorDescribeNodegroup: true},
		{name: "an error is logged for nodegroup list", ExpectErrorNodegroupList: true},
	}

	// Loop through each test case
	for _, c := range cases {
		// Construct a ListBucketsOutput object based on whether
		// we expect an error or not
		lcResponse := fakeEKSClustersSlice
		ldngRsponse := fakeEKSDescribeNodeGroup
		lngResponse := fakeEKSNodeGroupSlice

		switch {
		case c.ExpectErrorClusterList:
			lcResponse = nil
		case c.ExpectErrorDescribeNodegroup:
			ldngRsponse = nil
		case c.ExpectErrorNodegroupList:
			lngResponse = nil
		}

		// Create our fake service factory
		sf := fakeEKSServiceFactory{
			LCResponse:  lcResponse,
			DNGResponse: ldngRsponse,
			LNGResponse: lngResponse,
		}

		// Create a mock activity monitor
		mon := &mock.ActivityMonitorImpl{}

		t.Run(fmt.Sprintf("testing %s", c.name), func(t *testing.T) {
			// Invoke our EKS Function
			actualCount := EKSNodes(sf, mon, false)

			// Did we expect an error?
			if c.ExpectErrorNodegroupList || c.ExpectErrorClusterList || c.ExpectErrorDescribeNodegroup {
				// Did it fail to arrive?
				if !mon.ErrorOccured {
					t.Error("Expected an error to occur, but it did not... :^(")
				}
			} else if mon.ErrorOccured {
				t.Errorf("Unexpected error occurred: %s", mon.ErrorMessage)
			} else if actualCount != c.ExpectedCount {
				t.Errorf("Error: Nodes returned %d; expected %d", actualCount, c.ExpectedCount)
			} else if mon.ProgramExited {
				t.Errorf("Unexpected Exit: The program unexpected exited with status code=%d", mon.ExitCode)
			}
		})
	}
}
