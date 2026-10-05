package job

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"webhook/queue"
)

func Scheduler() {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		webhook := <-queue.WebhookQueue

		fmt.Println("Received job:", webhook.Message)

		resp, err := client.Get(webhook.CallBackUrl)
		if err != nil {
			log.Printf("error making request: %v", err)
			continue
		}

		resp.Body.Close()

		time.Sleep(10 * time.Second)

		fmt.Println("Finished job:", webhook.Message)
	}
}
