package main

import (
	"fmt"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/backends/null"
	gcppubsubbroker "github.com/RichardKnop/machinery/v2/brokers/gcppubsub"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
)

func main() {
	cfg := ceruviaConfig.GetAppConfig()

	brokerURL := fmt.Sprintf("gcppubsub://%s/%s", cfg.MQCfg.GCPProjectID, cfg.MQCfg.SubscriptionName)

	cnf := &config.Config{
		Broker:       brokerURL,
		DefaultQueue: cfg.MQCfg.TopicName,
	}

	broker, err := gcppubsubbroker.New(cnf, cfg.MQCfg.GCPProjectID, cfg.MQCfg.SubscriptionName)
	if err != nil {
		panic(err)
	}
	backend := null.New()
	lock := eagerlock.New()
	server := machinery.NewServer(cnf, broker, backend, lock)

	signature := &tasks.Signature{
		Name: "blackbox",
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: "test1",
			},
			{
				Type:  "string",
				Value: "https://pub-aa14e9fb26a94974a23c01cf74108727.r2.dev/c_adt_grader.zip",
			},
			{
				Type:  "string",
				Value: "https://pub-aa14e9fb26a94974a23c01cf74108727.r2.dev/c_adt_submission.zip",
			},
			{
				Type:  "[]string",
				Value: []string{"1.in", "2.in", "3.in", "4.in", "5.in", "6.in", "7.in", "8.in", "9.in", "10.in"},
			},
			{
				Type:  "[]string",
				Value: []string{"1.out", "2.out", "3.out", "4.out", "5.out", "6.out", "7.out", "8.out", "9.out", "10.out"},
			},
			{
				Type:  "int",
				Value: 1000,
			},
			{
				Type:  "int",
				Value: 10240,
			},
			{
				Type:  "string",
				Value: "C",
			},
			{
				Type:  "string",
				Value: "ganjilgenap.c",
			},
		},
	}

	asyncResult, err := server.SendTask(signature)
	if err != nil {
		panic(err)
	}

	fmt.Println("Task sent with UUID:", asyncResult.Signature.UUID)
}
