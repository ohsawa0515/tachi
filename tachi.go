package tachi

import (
	"context"
	"log"
	"time"
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

	clbClient := NewELBClient("clb", conf)
	if err := clbClient.FetchInstancesUnderLB(ctx, conf.Elbs); err != nil {
		log.Fatal(err)
	}

	albClient := NewELBClient("alb", conf)
	if err := albClient.FetchInstancesUnderLB(ctx, conf.Elbs); err != nil {
		log.Fatal(err)
	}

	elbClient := NewClient(clbClient, albClient, conf)
	if err := elbClient.RestartServers(); err != nil {
		log.Fatal(err)
	}

	return nil
}
