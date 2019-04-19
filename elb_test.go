package tachi

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

const (
	timeout  = 60
	coolDown = 2
	interval = 2
	waitTime = 2
	region   = "us-east-1"
)

var (
	conf Config
)

type mockElbSvc struct {
	elbiface.ELBAPI
}

type mockElbV2Svc struct {
	elbv2iface.ELBV2API
}

func (elbSvc *mockElbSvc) DescribeLoadBalancersWithContext(ctx aws.Context, input *elb.DescribeLoadBalancersInput, option ...request.Option) (*elb.DescribeLoadBalancersOutput, error) {
	return &elb.DescribeLoadBalancersOutput{}, nil
}

func (elbSvc *mockElbSvc) DescribeInstanceHealthWithContext(ctx aws.Context, input *elb.DescribeInstanceHealthInput, option ...request.Option) (*elb.DescribeInstanceHealthOutput, error) {
	return &elb.DescribeInstanceHealthOutput{
		InstanceStates: []*elb.InstanceState{
			{InstanceId: aws.String("i-12345678901234567"), State: aws.String("InService")},
			{InstanceId: aws.String("i-aaaaaaaaaaaaaaaaa"), State: aws.String("OutOfService")},
			{InstanceId: aws.String("i-bbbbbbbbbbbbbbbbb"), State: aws.String("Unknown")},
			{InstanceId: aws.String("i-ccccccccccccccccc"), State: aws.String("InService")},
		},
	}, nil
}

func (elbSvc *mockElbSvc) DeregisterInstancesFromLoadBalancer(input *elb.DeregisterInstancesFromLoadBalancerInput) (*elb.DeregisterInstancesFromLoadBalancerOutput, error) {
	return &elb.DeregisterInstancesFromLoadBalancerOutput{}, nil
}

func (elbSvc *mockElbSvc) WaitUntilInstanceDeregistered(input *elb.DescribeInstanceHealthInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}

func (elbSvc *mockElbSvc) RegisterInstancesWithLoadBalancer(input *elb.RegisterInstancesWithLoadBalancerInput) (*elb.RegisterInstancesWithLoadBalancerOutput, error) {
	return &elb.RegisterInstancesWithLoadBalancerOutput{}, nil
}

func (elbSvc *mockElbSvc) WaitUntilInstanceInService(input *elb.DescribeInstanceHealthInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}

func (elbV2Svc *mockElbV2Svc) DescribeLoadBalancersWithContext(ctx aws.Context, input *elbv2.DescribeLoadBalancersInput, option ...request.Option) (*elbv2.DescribeLoadBalancersOutput, error) {
	return &elbv2.DescribeLoadBalancersOutput{
		LoadBalancers: []*elbv2.LoadBalancer{
			{LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:ap-northeast-1:123456789012:loadbalancer/app/test-alb/1111111111111111")},
		},
	}, nil
}

func (elbV2Svc *mockElbV2Svc) DescribeTargetGroupsWithContext(ctx aws.Context, input *elbv2.DescribeTargetGroupsInput, option ...request.Option) (*elbv2.DescribeTargetGroupsOutput, error) {
	return &elbv2.DescribeTargetGroupsOutput{
		TargetGroups: []*elbv2.TargetGroup{
			{TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:ap-northeast-1:123456789012:targetgroup/test-alb/2222222222222222")},
		},
	}, nil
}

func (elbV2Svc *mockElbV2Svc) DescribeTargetHealthWithContext(ctx aws.Context, input *elbv2.DescribeTargetHealthInput, option ...request.Option) (*elbv2.DescribeTargetHealthOutput, error) {
	return &elbv2.DescribeTargetHealthOutput{
		TargetHealthDescriptions: []*elbv2.TargetHealthDescription{
			{Target: &elbv2.TargetDescription{Id: aws.String("i-12345678901234567")}, TargetHealth: &elbv2.TargetHealth{State: aws.String("healthy")}},
			{Target: &elbv2.TargetDescription{Id: aws.String("i-aaaaaaaaaaaaaaaaa")}, TargetHealth: &elbv2.TargetHealth{State: aws.String("unhealthy")}},
			{Target: &elbv2.TargetDescription{Id: aws.String("i-bbbbbbbbbbbbbbbbb")}, TargetHealth: &elbv2.TargetHealth{State: aws.String("unused")}},
			{Target: &elbv2.TargetDescription{Id: aws.String("i-ccccccccccccccccc")}, TargetHealth: &elbv2.TargetHealth{State: aws.String("healthy")}},
		},
	}, nil
}

func (elbV2Svc *mockElbV2Svc) DeregisterTargets(input *elbv2.DeregisterTargetsInput) (*elbv2.DeregisterTargetsOutput, error) {
	return &elbv2.DeregisterTargetsOutput{}, nil
}

func (elbV2Svc *mockElbV2Svc) WaitUntilTargetDeregistered(input *elbv2.DescribeTargetHealthInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}

func (elbV2Svc *mockElbV2Svc) RegisterTargets(input *elbv2.RegisterTargetsInput) (*elbv2.RegisterTargetsOutput, error) {
	return &elbv2.RegisterTargetsOutput{}, nil
}

func (elbV2Svc *mockElbV2Svc) WaitUntilTargetInService(*elbv2.DescribeTargetHealthInput) error {
	time.Sleep(time.Duration(waitTime * time.Second))
	return nil
}

func clbFetchInstancesUnderLB(ctx context.Context, conf Config) (LBiface, error) {
	elbSvc := &mockElbSvc{}
	clbClient := NewClbClient(elbSvc)
	if err := clbClient.FetchInstancesUnderLB(ctx, conf); err != nil {
		return nil, err
	}
	return clbClient, nil
}

func albFetchInstancesUnderLB(ctx context.Context, conf Config) (LBiface, error) {
	elbV2Svc := &mockElbV2Svc{}
	albClient := NewAlbClient(elbV2Svc)
	if err := albClient.FetchInstancesUnderLB(ctx, conf); err != nil {
		return nil, err
	}
	return albClient, nil
}

func TestRestartServers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	clbClient, err := clbFetchInstancesUnderLB(ctx, conf)
	if err != nil {
		t.Error(err)
	}
	albClient, err := albFetchInstancesUnderLB(ctx, conf)
	if err != nil {
		t.Error(err)
	}

	elbClient := NewClient(ec2Svc, ssmSvc, clbClient, albClient)
	if err := elbClient.RestartServers(conf); err != nil {
		t.Error(err)
	}
}

// main //
func setup() {
	elbs := []string{"test-clb", "test-alb"}
	conf = Config{
		Elbs:     elbs,
		Timeout:  time.Duration(timeout) * time.Second,
		CoolDown: time.Duration(coolDown) * time.Second,
		Interval: time.Duration(interval) * time.Second,
		Region:   region,
		Logger:   NewLogger(ioutil.Discard), // do not output
	}

	ec2Svc = &mockEC2Svc{}
	ssmSvc = &mockSsmSvc{}
}

func TestMain(m *testing.M) {
	setup()
	exitCode := m.Run()
	os.Exit(exitCode)
}
