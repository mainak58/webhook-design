package job

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"webhook/queue"
)

var failedResponse = make(chan queue.WebHook, 1000)
var maxRetries = 3

var (
	successMu       sync.Mutex
	successResponse = []string{}
)

func Scheduler() {

	reqNumber := 0

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		webhook := <-queue.WebhookQueue
		reqNumber++

		success := false
		fmt.Println("Received job:", webhook.Message)

		for i := 0; i <= maxRetries; i++ {
			resp, err := client.Get(webhook.CallBackUrl)
			if err != nil {
				log.Printf("job %d attempt %d: error making request: %v", reqNumber, i+1, err)
				time.Sleep(10 * time.Second)
				continue
			}
			resp.Body.Close()
			if resp.StatusCode < 300 {
				success = true
				break
			}

			log.Printf("job %d attempt %d: bad status %d", reqNumber, i+1, resp.StatusCode)
			time.Sleep(10 * time.Second)
		}

		if success {
			successMu.Lock()
			successResponse = append(successResponse, webhook.Message)
			successMu.Unlock()
		} else {
			failedResponse <- webhook
		}

		fmt.Println("Finished job:", webhook.Message)
	}
}
