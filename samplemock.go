package main

import (
	"testing"
	"github.com/stretchr/testify/mock"
)

// Create a mock struct for the KeployDeployer interface
type MockKeployDeployer struct {
	mock.Mock
}

// Implement the Deploy method of the KeployDeployer interface for the mock struct
func (m *MockKeployDeployer) Deploy(image string) error {
	args := m.Called(image)
	return args.Error(0)
}

func TestDeployWithKeploy(t *testing.T) {
	// Create an instance of the mock struct
	mockDeployer := new(MockKeployDeployer)

	// Set up expectations for the Deploy method
	mockDeployer.On("Deploy", "my-image").Return(nil)

	// Call the function that uses the KeployDeployer interface
	err := deployWithKeploy(mockDeployer, "my-image")

	// Assert that the mock object was called as expected
	mockDeployer.AssertExpectations(t)

	// Assert that no error occurred during the deployment
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// The function that uses the KeployDeployer interface
func deployWithKeploy(deployer KeployDeployer, image string) error {
	// Call the Deploy method of the KeployDeployer interface
	err := deployer.Deploy(image)
	return err
}

// Define the KeployDeployer interface
type KeployDeployer interface {
	Deploy(image string) error
}