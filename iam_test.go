package main

import (
    "testing"

    "github.com/aws/aws-sdk-go/service/iam"
    "github.com/aws/aws-sdk-go/service/iam/iamiface"
    "github.com/stretchr/testify/mock"
)

// MockIAMAPI is a mock of the IAMAPI interface
type MockIAMAPI struct {
    mock.Mock
    iamiface.IAMAPI
}

// ListUsersPages mocks the ListUsersPages method
func (m *MockIAMAPI) ListUsersPages(input *iam.ListUsersInput, fn func(*iam.ListUsersOutput, bool) bool) error {
    args := m.Called(input, fn)
    output := &iam.ListUsersOutput{
        Users: []*iam.User{
            {UserName: aws.String("user1")},
            {UserName: aws.String("user2")},
        },
    }
    fn(output, true)
    return args.Error(0)
}

func TestIAMUserCounts(t *testing.T) {
    mockIAM := new(MockIAMAPI)
    mockIAM.On("ListUsersPages", mock.Anything, mock.Anything).Return(nil)

    sf := &AWSServiceFactory{Session: nil} 
    sf.IAMService = &IAMService{Client: mockIAM}

    am := &mockActivityMonitor{}

    count := IAMUserCounts(sf, am)
    if count != 2 {
        t.Errorf("Expected 2 IAM users, got %d", count)
    }
}