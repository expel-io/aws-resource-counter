package main

import (
    "github.com/aws/aws-sdk-go/service/iam"

	color "github.com/logrusorgru/aurora"

)

// IAMUserCounts retrieves the count of IAM users.
func IAMUserCounts(sf ServiceFactory, am ActivityMonitor) int {
    am.StartAction("Retrieving IAM User counts")

    iamService := sf.GetIAMService()
    input := &iam.ListUsersInput{}

    // Initialize user count
    userCount := 0

    // Paginate through the list of IAM users
    err := iamService.Client.ListUsersPages(input, func(page *iam.ListUsersOutput, lastPage bool) bool {
        userCount += len(page.Users)
        return !lastPage
    })

    if err != nil {
        am.CheckError(err)
        return 0
    }

	am.EndAction("OK (%d)", color.Bold(userCount))

    return userCount
}