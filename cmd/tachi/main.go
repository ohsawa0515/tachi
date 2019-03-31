package main

import (
	"flag"
	"strings"
	"time"

	"github.com/ohsawa0515/tachi"
)

var (
	elbs     string
	timeout  int
	coolDown int
	interval int
	region   string
)

func main() {

	flag.StringVar(&elbs, "elbs", "", "ELB's name. Comma separated. e.g. test-clb,test-alb")
	flag.IntVar(&timeout, "timeout", 60, "Timeout for calling AWS API")
	flag.IntVar(&coolDown, "cooldown", 60, "Period from EC2 instance startup to normal handling")
	flag.IntVar(&interval, "interval", 60, "Interval from the attachment of an ELB to the detachment of the next EC2 instance")
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.Parse()

	log := tachi.NewLogger()
	conf := tachi.Config{
		Elbs:     strings.Split(elbs, ","),
		Timeout:  time.Duration(timeout) * time.Second,
		CoolDown: time.Duration(coolDown) * time.Second,
		Interval: time.Duration(interval) * time.Second,
		Region:   region,
		Logger:   log,
	}

	if err := tachi.Run(conf); err != nil {
		log.Fatalf("%+v\n", err)
	}
}
