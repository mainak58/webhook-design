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

// 1 attempt in Scheduler + 4 in pickUpFailedJob = 5 total before permanentResponse.
const (
	firstAttempts = 1
	retryAttempts = 4
	retryDelay    = 1 * time.Minute
)

var (
	successMu       sync.Mutex
	successResponse = []string{}
)

var (
	permanentMu       sync.Mutex
	permanentResponse = []string{}
)

func deliver(client *http.Client, webhook queue.WebHook, attempts int) bool {
	for i := 0; i < attempts; i++ {
		resp, err := client.Get(webhook.CallBackUrl)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 300 {
				return true
			}
			log.Printf("%s attempt %d: bad status %d", webhook.Message, i+1, resp.StatusCode)
		} else {
			log.Printf("%s attempt %d: error making request: %v", webhook.Message, i+1, err)
		}
		if i < attempts-1 {
			time.Sleep(10 * time.Second)
		}
	}
	return false
}

func Scheduler() {
	go pickUpFailedJob()

	reqNumber := 0

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		webhook := <-queue.WebhookQueue
		reqNumber++

		fmt.Println("Received job", reqNumber, ":", webhook.Message)

		if deliver(client, webhook, firstAttempts) {
			successMu.Lock()
			successResponse = append(successResponse, webhook.Message)
			successMu.Unlock()
		} else {
			failedResponse <- webhook
		}

		fmt.Println("Finished job", reqNumber, ":", webhook.Message)
	}
}

func pickUpFailedJob() {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		webhook := <-failedResponse

		time.Sleep(retryDelay)
		fmt.Println("Retrying job:", webhook.Message)

		if deliver(client, webhook, retryAttempts) {
			successMu.Lock()
			successResponse = append(successResponse, webhook.Message)
			successMu.Unlock()
		} else {
			permanentMu.Lock()
			permanentResponse = append(permanentResponse, webhook.Message)
			permanentMu.Unlock()
		}

		fmt.Println("Finished retry:", webhook.Message)
	}
}
