package main

import (
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/ohsawa0515/tachi"
)

var (
	elbs             string
	mode             string
	command          string
	documentName     string
	executionTimeout int
	timeout          int
	coolDown         int
	interval         int
	region           string
)

func main() {

	flag.StringVar(&elbs, "-elbs", "", "ELB's name. Comma separated. e.g. test-clb,test-alb")
	flag.StringVar(&mode, "-mode", tachi.ModeRboot, "Action mode for EC2. Vaild values are 'reboot', 'ssm' (Systems Manager Run Command)")
	flag.StringVar(&documentName, "-document-name", tachi.RunShellScript, "The name of the Systems Manager document to run. Valid values are 'AWS-RunShellScript', 'AWS-RunPowerShellScript'. ssm mode only")
	flag.StringVar(&command, "-command", "", "Command to execute on EC2. ssm mode only")
	flag.IntVar(&executionTimeout, "-execution-timeout", 300, "Execution timeout")
	flag.IntVar(&timeout, "-timeout", 60, "Timeout for calling AWS API")
	flag.IntVar(&coolDown, "-cooldown", 60, "Period from EC2 instance startup to normal handling")
	flag.IntVar(&interval, "-interval", 60, "Interval from the attachment of an ELB to the detachment of the next EC2 instance")
	flag.StringVar(&region, "-region", "ap-northeast-1", "Region")
	flag.Parse()

	log := tachi.NewLogger()
	conf := tachi.Config{
		Elbs:             strings.Split(elbs, ","),
		Mode:             mode,
		Command:          command,
		DocumentName:     documentName,
		ExecutionTimeout: strconv.Itoa(executionTimeout),
		Timeout:          time.Duration(timeout) * time.Second,
		CoolDown:         time.Duration(coolDown) * time.Second,
		Interval:         time.Duration(interval) * time.Second,
		Region:           region,
		Logger:           log,
	}

	if err := tachi.Run(conf); err != nil {
		log.Fatalf("%+v\n", err)
	}
}
