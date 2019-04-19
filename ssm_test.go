package tachi

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

type mockSsmSvc struct {
	ssmiface.SSMAPI
}

func (ssmSvc *mockSsmSvc) DescribeInstanceInformation(*ssm.DescribeInstanceInformationInput) (*ssm.DescribeInstanceInformationOutput, error) {
	return &ssm.DescribeInstanceInformationOutput{
		InstanceInformationList: []*ssm.InstanceInformation{
			{PingStatus: aws.String("Online")},
		},
	}, nil
}

func (ssmSvc *mockSsmSvc) SendCommand(*ssm.SendCommandInput) (*ssm.SendCommandOutput, error) {
	return &ssm.SendCommandOutput{
		Command: &ssm.Command{
			CommandId: aws.String("123456789"),
		},
	}, nil
}

func (ssmSvc *mockSsmSvc) GetCommandInvocation(*ssm.GetCommandInvocationInput) (*ssm.GetCommandInvocationOutput, error) {
	return &ssm.GetCommandInvocationOutput{
		Status: aws.String("Success"),
	}, nil
}

func TestExecuteSsmRunCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	elbs := []string{"test-clb", "test-alb"}
	conf := Config{
		Elbs:     elbs,
		Timeout:  time.Duration(timeout) * time.Second,
		CoolDown: time.Duration(coolDown) * time.Second,
		Interval: time.Duration(interval) * time.Second,
		Region:   region,
		Logger:   NewLogger(),
	}

	clbClient, err := clbFetchInstancesUnderLB(ctx, conf)
	if err != nil {
		t.Error(err)
	}
	albClient, err := albFetchInstancesUnderLB(ctx, conf)
	if err != nil {
		t.Error(err)
	}

	log.Println(clbClient.Servers())
	log.Println(albClient.Servers())

	ec2Svc := &mockEC2Svc{}
	ssmSvc := &mockSsmSvc{}
	elbClient := NewClient(ec2Svc, ssmSvc, clbClient, albClient)
	if err := elbClient.ExecuteSsmRunCommand(conf); err != nil {
		t.Error(err)
	}
}
