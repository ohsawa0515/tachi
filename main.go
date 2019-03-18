package main

import (
	"context"
	"log"
	"time"
)

var (
	region = "ap-northeast-1"
)

const (
	timeout  = 30
	cooldown = 60
	interval = 60
)

func main() {
	elbs := []string{"test-clb", "test-alb"}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	clbClient := NewELBClient("clb", region)
	if err := clbClient.FetchInstancesUnderLB(ctx, elbs...); err != nil {
		log.Fatal(err)
	}

	albClient := NewELBClient("alb", region)
	if err := albClient.FetchInstancesUnderLB(ctx, elbs...); err != nil {
		log.Fatal(err)
	}

	elbClient := NewClient(clbClient, albClient)
	if err := elbClient.RestartServers(); err != nil {
		log.Fatal(err)
	}

}
