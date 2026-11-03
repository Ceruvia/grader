// Command send publishes one sample grading task through the broker QUEUE_BROKER
// names. The result arrives at BACKEND_CALLBACK_ENDPOINT, not here.
package main

import (
	"fmt"
	"log"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/RichardKnop/machinery/v2/tasks"
)

func main() {
	cfg, err := ceruviaConfig.GetAppConfig()
	if err != nil {
		log.Fatal(err)
	}

	server, err := machinery.NewServer(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}

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

	if _, err := server.SendTask(signature); err != nil {
		log.Fatal(err)
	}
	fmt.Println("sent", signature.UUID)
}
