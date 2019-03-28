package tachi

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

type Config struct {
	Elbs     []string
	Timeout  time.Duration
	CoolDown time.Duration
	Interval time.Duration
	Region   string
}

// Run -
func Run(conf Config) error {

	ctx, cancel := context.WithTimeout(context.Background(), conf.Timeout)
	defer cancel()

	sess := session.Must(session.NewSession(
		&aws.Config{Region: aws.String(conf.Region)}))

	clbClient := NewClbClient(elb.New(sess))
	if err := clbClient.FetchInstancesUnderLB(ctx, conf.Elbs); err != nil {
		log.Fatal(err)
	}

	albClient := NewAlbClient(elbv2.New(sess))
	if err := albClient.FetchInstancesUnderLB(ctx, conf.Elbs); err != nil {
		log.Fatal(err)
	}

	elbClient := NewClient(ec2.New(sess), clbClient, albClient)
	if err := elbClient.RestartServers(); err != nil {
		log.Fatal(err)
	}

	return nil
}
