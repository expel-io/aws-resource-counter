/******************************************************************************
Cloud Resource Counter
File: s3_test.go

Summary: The Unit Test for s3.
******************************************************************************/

package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/eks/eksiface"
	"github.com/expel-io/aws-resource-counter/mock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
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

var fakeEKSDescribeCluster = &eks.DescribeClusterOutput{
	Cluster: &eks.Cluster{
		Name:     aws.String("cluster"),
		Endpoint: aws.String("endpoint-string"),
	},
}

var fakeNodes = []*v1.Node{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "node-1",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:        "node-2",
			Namespace:   "default",
			Annotations: map[string]string{},
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
	LDCResponse *eks.DescribeClusterOutput
}

func (feks *fakeEKService) DescribeCluster(input *eks.DescribeClusterInput) (*eks.DescribeClusterOutput, error) {
	// If there was no supplied response, then simulate a possible error
	if feks.LDCResponse == nil {
		return nil, errors.New("ListClusters returns an unexpected error: 2345")
	}

	return feks.LDCResponse, nil
}

// Simulate the ListClustersPagesfunction
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

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Fake Service Factory
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

// This structure simulates the AWS Service Factory by storing some pregenerated
// responses (that would come from AWS).
type fakeEKSServiceFactory struct {
	LCResponse  []*eks.ListClustersOutput
	LDCResponse *eks.DescribeClusterOutput
	Nodes       []*v1.Node
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
			LDCResponse: fsf.LDCResponse,
		},
	}
}

func (fsf fakeEKSServiceFactory) GetK8Service(cf ClusterFactory, clusterEndpoint string) *K8Service {
	if fsf.Nodes == nil {
		return nil
	}
	return &K8Service{
		Client: testclient.NewSimpleClientset(fsf.Nodes[0], fsf.Nodes[1]),
	}
}

// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=
// Unit Test for S3Buckets
// =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

func TestEKSNodes(t *testing.T) {
	// Describe all of our test cases: 1 failure and 1 success
	cases := []struct {
		ExpectedCount              int
		ExpectErrorClusterList     bool
		ExpectErrorDescribeCluster bool
		ExpectErrorK8Client        bool
	}{
		// Expected count is 6 because there are 2 nodes defined for each cluster
		{ExpectedCount: 6},
		{ExpectErrorClusterList: true},
		{ExpectErrorDescribeCluster: true},
		{ExpectErrorK8Client: true},
	}

	// Loop through each test case
	for _, c := range cases {
		// Construct a ListBucketsOutput object based on whether
		// we expect an error or not
		lcResponse := fakeEKSClustersSlice
		ldcRsponse := fakeEKSDescribeCluster
		nodes := fakeNodes

		switch {
		case c.ExpectErrorClusterList:
			lcResponse = nil
		case c.ExpectErrorDescribeCluster:
			ldcRsponse = nil
		case c.ExpectErrorK8Client:
			nodes = nil
		}

		// Create our fake service factory
		sf := fakeEKSServiceFactory{
			LCResponse:  lcResponse,
			LDCResponse: ldcRsponse,
			Nodes:       nodes,
		}

		// Create a mock activity monitor
		mon := &mock.ActivityMonitorImpl{}

		// Invoke our EKS Function
		actualCount := EKSNodes(sf, mon, false)

		// Did we expect an error?
		if c.ExpectErrorK8Client || c.ExpectErrorClusterList || c.ExpectErrorDescribeCluster {
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
	}
}
