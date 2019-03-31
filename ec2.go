package tachi

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
)

func (c *Client) restartServer(server Server, conf Config) error {

	conf.Logger.Infof("Instance %s will restart from now on", server.id)

	// Stop instance
	if _, err := c.ec2Svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(server.id),
		},
		Force: aws.Bool(true),
	}); err != nil {
		return errors.WithStack(err)
	}
	if err := c.ec2Svc.WaitUntilInstanceStopped(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(server.id),
		},
	}); err != nil {
		return errors.WithStack(err)
	}

	// Start instance
	if _, err := c.ec2Svc.StartInstances(&ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(server.id),
		},
	}); err != nil {
		return errors.WithStack(err)
	}
	if err := c.ec2Svc.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{
			aws.String(server.id),
		},
	}); err != nil {
		return errors.WithStack(err)
	}

	// Wait until cool down time
	time.Sleep(conf.CoolDown)

	conf.Logger.Infof("Instance %s has been restartd", server.id)

	return nil
}
