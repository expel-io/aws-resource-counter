/******************************************************************************
Cloud Resource Counter
File: utils.go

Summary: Various utility functions
******************************************************************************/

package main

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// OpenFileForWriting does stuff...
func OpenFileForWriting(fileName string, typeOfFile string, am ActivityMonitor, append bool) *os.File {
	// What is our flag for the file?
	var flag int
	if append {
		flag = os.O_WRONLY | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	// Can we open it for writing?
	file, err := os.OpenFile(fileName, flag, 0666)

	// Check for error
	am.CheckError(err)

	return file
}

// GetEC2Regions determines the set of regions associated with the account.
func GetEC2Regions(ec2is *EC2InstanceService, am ActivityMonitor) []string {
	// Construct the input
	input := &ec2.DescribeRegionsInput{
		Filters: []types.Filter{
			{
				Name: aws.String("opt-in-status"),
				Values: []string{
					"opt-in-not-required",
					"opted-in",
				},
			},
		},
	}

	// Execute the command
	result, err := ec2is.GetRegions(input)

	// Do we have an error?
	if am.CheckError(err) {
		return nil
	}

	// Transform the array of results into an array of region names...
	var regionNames []string
	for _, regionInfo := range result.Regions {
		regionNames = append(regionNames, *regionInfo.RegionName)
	}

	return regionNames
}

// IsValidRegionName returns whether the supplied region name is valid or not.
func IsValidRegionName(ctx context.Context, regionName string) (bool, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return false, err
	}

	svc := ec2.NewFromConfig(cfg)

	resp, err := svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return false, fmt.Errorf("failed to describe regions, %v", err)
	}

	// Loop through the region names...
	for _, region := range resp.Regions {
		// Does it match?
		if *region.RegionName == regionName {
			return true, nil
		}
	}

	return false, nil
}

// Map applies a function to each element of a string array
// Borrowed from: https://gobyexample.com/collection-functions
func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

// NilInterface checks whether the supplied interface is nil or not
func NilInterface(intf interface{}) bool {
	return intf == nil || reflect.ValueOf(intf).IsNil()
}

// FileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func FileExists(fileName string) bool {
	// Stat the file
	info, err := os.Stat(fileName)

	// Check for non-existent file
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}
