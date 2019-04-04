package tachi

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"
)

const (
	RunShellScript      = "AWS-RunShellScript"
	RunPowerShellScript = "AWS-RunPowerShellScript"
	waitExecution       = 15
	timeoutSecond       = 600
)

func (c *Client) sendCommand(server Server, conf Config) error {

	conf.Logger.Infof("Instance %s will execute ssm from now on", server.id)

	i, err := c.ssmSvc.DescribeInstanceInformation(&ssm.DescribeInstanceInformationInput{
		InstanceInformationFilterList: []*ssm.InstanceInformationFilter{
			{
				Key: aws.String("InstanceIds"),
				ValueSet: []*string{
					aws.String(server.id),
				},
			},
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if len(i.InstanceInformationList) == 0 {
		return errors.Errorf("SSM agent of instance %s has not been started or has not been activated", server.id)
	}

	if *i.InstanceInformationList[0].PingStatus != "Online" {
		return errors.Errorf("Instance %s does not online on SSM", server.id)
	}

	resp, err := c.ssmSvc.SendCommand(&ssm.SendCommandInput{
		DocumentName: aws.String(conf.DocumentName),
		InstanceIds: []*string{
			aws.String(server.id),
		},
		Parameters: map[string][]*string{
			"commands":         {aws.String(conf.Command)},
			"executionTimeout": {aws.String(conf.ExecutionTimeout)},
		},
		TimeoutSeconds: aws.Int64(timeoutSecond),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	conf.Logger.Infof("Waiting for command execution of instance %s to complete", server.id)
	time.Sleep(time.Duration(waitExecution) * time.Second)

	resultCode := false
L:
	for {
		result, err := c.ssmSvc.GetCommandInvocation(&ssm.GetCommandInvocationInput{
			CommandId:  resp.Command.CommandId,
			InstanceId: aws.String(server.id),
		})
		if err != nil {
			return errors.WithStack(err)
		}
		switch *result.Status {
		case "Success":
			resultCode = true
			break L
		case "Delivery Timed Out", "Execution Timed Out", "Failed", "Canceled", "Undeliverable", "Terminated":
			resultCode = false
			break L
		case "Pending", "In Progress":
			time.Sleep(time.Duration(waitExecution) * time.Second)
		default:
			resultCode = false
			break L
		}
	}

	if !resultCode {
		return errors.Errorf("Execution instance %s failed", server.id)
	}

	// Wait until cool down time
	time.Sleep(conf.CoolDown)
	conf.Logger.Infof("Instance %s has been executed", server.id)

	return nil
}
