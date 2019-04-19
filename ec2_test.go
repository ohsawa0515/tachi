package tachi

import (
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
)

var (
	ec2Svc ec2iface.EC2API
)

type mockEC2Svc struct {
	ec2iface.EC2API
}

func (ec2Svc *mockEC2Svc) StopInstances(input *ec2.StopInstancesInput) (*ec2.StopInstancesOutput, error) {
	return &ec2.StopInstancesOutput{}, nil
}

func (ec2Svc *mockEC2Svc) WaitUntilInstanceStopped(input *ec2.DescribeInstancesInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}

func (ec2Svc *mockEC2Svc) StartInstances(*ec2.StartInstancesInput) (*ec2.StartInstancesOutput, error) {
	return &ec2.StartInstancesOutput{}, nil
}

func (ec2Svc *mockEC2Svc) WaitUntilInstanceStatusOk(*ec2.DescribeInstanceStatusInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}
