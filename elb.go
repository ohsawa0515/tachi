package tachi

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// LBiface -
type LBiface interface {
	FetchInstancesUnderLB(context.Context, Config) error
	Servers() Servers
	Lbs() []string
	ELbSvc() elbiface.ELBAPI
	ELbV2Svc() elbv2iface.ELBV2API
}

// Client is common client of clb and alb
type Client struct {
	ec2Svc    ec2iface.EC2API
	clbClient LBiface
	albClient LBiface
	servers   Servers
}

// clbClient -
type clbClient struct {
	elbSvc        elbiface.ELBAPI
	loadBalancers []string
	servers       Servers
}

// albClient -
type albClient struct {
	elbv2Svc        elbv2iface.ELBV2API
	targetGroupArns []string
	servers         Servers
}

// Servers -
type Servers []Server

// Server is EC2 instance
type Server struct {
	id string
}

// NewClbClient -
func NewClbClient(svc elbiface.ELBAPI) LBiface {
	return &clbClient{
		elbSvc: svc,
	}
}

// NewAlbClient -
func NewAlbClient(svc elbv2iface.ELBV2API) LBiface {
	return &albClient{
		elbv2Svc: svc,
	}
}

// NewClient -
func NewClient(svc ec2iface.EC2API, clbClient LBiface, albClient LBiface) *Client {

	// Merge servers
	servers := Servers{}
	m := make(map[string]struct{})
	for _, server := range clbClient.Servers() {
		// Double check
		if _, ok := m[server.id]; !ok {
			servers = append(servers, server)
			m[server.id] = struct{}{}
		}
	}
	for _, server := range albClient.Servers() {
		// Double check
		if _, ok := m[server.id]; !ok {
			servers = append(servers, server)
			m[server.id] = struct{}{}
		}
	}

	return &Client{
		ec2Svc:    svc,
		clbClient: clbClient,
		albClient: albClient,
		servers:   servers,
	}
}

// FetchInstancesUnderLB returns instances belonging to Classic Load Balancers
func (c *clbClient) FetchInstancesUnderLB(ctx context.Context, conf Config) error {
	eg := errgroup.Group{}
	m := sync.Map{}
	for _, clb := range conf.Elbs {
		clb := clb
		eg.Go(func() error {
			if _, err := c.elbSvc.DescribeLoadBalancersWithContext(ctx, &elb.DescribeLoadBalancersInput{
				LoadBalancerNames: []*string{
					aws.String(clb),
				},
			}); err != nil {
				conf.Logger.Debug(err)
				return nil // Not Found
			}

			// append load balancer
			c.loadBalancers = append(c.loadBalancers, clb)

			resp, err := c.elbSvc.DescribeInstanceHealthWithContext(ctx, &elb.DescribeInstanceHealthInput{
				LoadBalancerName: aws.String(clb),
			})
			if err != nil {
				return err
			}

			for _, i := range resp.InstanceStates {
				if *i.State == "InService" {
					if _, ok := m.Load(*i.InstanceId); !ok { // 重複チェック
						c.servers = append(c.servers, Server{id: *i.InstanceId})
						m.Store(*i.InstanceId, struct{}{}) // インスタンスIDを登録、重複チェックに利用する
					}
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// FetchInstancesUnderLB returns instances belonging to Application Load Balancers
func (a *albClient) FetchInstancesUnderLB(ctx context.Context, conf Config) error {
	eg := errgroup.Group{}
	m := sync.Map{}
	for _, alb := range conf.Elbs {
		alb := alb
		eg.Go(func() error {
			arn, err := a.GetTargetGroupArnFromLoadBalancerName(ctx, alb)
			if err != nil {
				conf.Logger.Debug(err)
				return nil // Not Found
			}

			// append loadbalancer
			a.targetGroupArns = append(a.targetGroupArns, *arn)

			resp, err := a.elbv2Svc.DescribeTargetHealthWithContext(ctx, &elbv2.DescribeTargetHealthInput{
				TargetGroupArn: arn,
			})
			if err != nil {
				return err
			}

			for _, t := range resp.TargetHealthDescriptions {
				if *t.TargetHealth.State == "healthy" {
					if _, ok := m.Load(*t.Target.Id); !ok { // 重複チェック
						a.servers = append(a.servers, Server{id: *t.Target.Id})
						m.Store(*t.Target.Id, struct{}{}) // インスタンスIDを登録、重複チェックに利用する
					}
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// Servers return information of ec2 instances
func (c *clbClient) Servers() Servers {
	return c.servers
}

// Servers return information of ec2 instances
func (a *albClient) Servers() Servers {
	return a.servers
}

// Lbs return name of classic load balancers
func (c *clbClient) Lbs() []string {
	return c.loadBalancers
}

// Lbs return name of application load balancer arn
func (a *albClient) Lbs() []string {
	return a.targetGroupArns
}

// GetElbSvc -
func (c *clbClient) ELbSvc() elbiface.ELBAPI {
	return c.elbSvc
}

// GetElbV2Svc -
func (c *clbClient) ELbV2Svc() elbv2iface.ELBV2API {
	return nil
}

// GetElbSvc -
func (a *albClient) ELbSvc() elbiface.ELBAPI {
	return nil
}

// GetElbV2Svc -
func (a *albClient) ELbV2Svc() elbv2iface.ELBV2API {
	return a.elbv2Svc
}

// RestartServers reboots the servers.
// When rebooting, detach from the ELB and attach to the ELB when rebooting is complete.
func (c *Client) RestartServers(conf Config) error {

	for _, server := range c.servers {
		if err := c.detachFromLoadBalancer(server, conf); err != nil {
			return err
		}

		if err := c.restartServer(server, conf); err != nil {
			return err
		}

		if err := c.attachWithLoadBalancer(server, conf); err != nil {
			return err
		}
	}

	return nil
}

// GetApplicationLoadBalancerArn returns arn of application loadbalancer giving loadbalancer name
func (a *albClient) GetApplicationLoadBalancerArn(ctx context.Context, alb string) (*string, error) {
	resp, err := a.elbv2Svc.DescribeLoadBalancersWithContext(ctx, &elbv2.DescribeLoadBalancersInput{
		Names: []*string{
			aws.String(alb),
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.LoadBalancers[0].LoadBalancerArn, nil
}

// GetTargetGroupArnFromLoadBalancerName returns tareget group arn of application loadbalancer giving loadbalancer name
func (a *albClient) GetTargetGroupArnFromLoadBalancerName(ctx context.Context, alb string) (*string, error) {
	arn, err := a.GetApplicationLoadBalancerArn(ctx, alb)
	if err != nil {
		return nil, err
	}

	resp, err := a.elbv2Svc.DescribeTargetGroupsWithContext(ctx, &elbv2.DescribeTargetGroupsInput{
		LoadBalancerArn: arn,
	})
	if err != nil {
		return nil, err
	}

	return resp.TargetGroups[0].TargetGroupArn, nil
}

func (c *Client) detachFromLoadBalancer(server Server, conf Config) error {

	conf.Logger.Infof("Instance %s will detach from ELB", server.id)

	eg := errgroup.Group{}

	// Classic load balancer
	for _, clb := range c.clbClient.Lbs() {
		clb := clb
		eg.Go(func() error {
			if _, err := c.clbClient.ELbSvc().DeregisterInstancesFromLoadBalancer(&elb.DeregisterInstancesFromLoadBalancerInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(server.id),
					},
				},
				LoadBalancerName: aws.String(clb),
			}); err != nil {
				return err
			}

			return c.clbClient.ELbSvc().WaitUntilInstanceDeregistered(&elb.DescribeInstanceHealthInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(server.id),
					},
				},
				LoadBalancerName: aws.String(clb),
			})
		})
	}

	// Application load balancer
	for _, alb := range c.albClient.Lbs() {
		alb := alb
		eg.Go(func() error {
			if _, err := c.albClient.ELbV2Svc().DeregisterTargets(&elbv2.DeregisterTargetsInput{
				Targets: []*elbv2.TargetDescription{
					{
						Id: aws.String(server.id),
					},
				},
				TargetGroupArn: aws.String(alb),
			}); err != nil {
				return err
			}

			return c.albClient.ELbV2Svc().WaitUntilTargetDeregistered(&elbv2.DescribeTargetHealthInput{
				Targets: []*elbv2.TargetDescription{
					{
						Id: aws.String(server.id),
					},
				},
				TargetGroupArn: aws.String(alb),
			})
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	conf.Logger.Infof("Instance %s has been detached from ELB", server.id)

	return nil
}

func (c *Client) attachWithLoadBalancer(server Server, conf Config) error {

	conf.Logger.Infof("Instance %s will attach to ELB from now on", server.id)

	eg := errgroup.Group{}

	for _, clb := range c.clbClient.Lbs() {
		eg.Go(func() error {
			if _, err := c.clbClient.ELbSvc().RegisterInstancesWithLoadBalancer(&elb.RegisterInstancesWithLoadBalancerInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(server.id),
					},
				},
				LoadBalancerName: aws.String(clb),
			}); err != nil {
				return err
			}

			return c.clbClient.ELbSvc().WaitUntilInstanceInService(&elb.DescribeInstanceHealthInput{
				Instances: []*elb.Instance{
					{
						InstanceId: aws.String(server.id),
					},
				},
				LoadBalancerName: aws.String(clb),
			})
		})
	}

	for _, alb := range c.albClient.Lbs() {
		eg.Go(func() error {
			if _, err := c.albClient.ELbV2Svc().RegisterTargets(&elbv2.RegisterTargetsInput{
				Targets: []*elbv2.TargetDescription{
					{
						Id: aws.String(server.id),
					},
				},
				TargetGroupArn: aws.String(alb),
			}); err != nil {
				return err
			}

			return c.albClient.ELbV2Svc().WaitUntilTargetInService(&elbv2.DescribeTargetHealthInput{
				Targets: []*elbv2.TargetDescription{
					{
						Id: aws.String(server.id),
					},
				},
				TargetGroupArn: aws.String(alb),
			})
		})
	}

	if err := eg.Wait(); err != nil {
		return errors.WithStack(err)
	}

	conf.Logger.Infof("Instance %s has been attached to ELB", server.id)

	// Wait until interval
	time.Sleep(conf.Interval)

	return nil
}
