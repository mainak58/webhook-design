package job

import (
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
	"webhook/queue"
)

var (
	successMu       sync.Mutex
	successResponse = []string{}
)

var (
	permanentMu       sync.Mutex
	permanentResponse = []string{}
)

func recordDead(wh queue.WebHook) {

	permanentMu.Lock()
	permanentResponse = append(permanentResponse, wh.Message)
	permanentMu.Unlock()

}

func recordDone(wh queue.WebHook) {

	successMu.Lock()
	successResponse = append(successResponse, wh.Message)
	successMu.Unlock()

}

func deliverOnce(client *http.Client, wh queue.WebHook) bool {
	resp, err := client.Get(wh.CallBackUrl)
	if err != nil {
		log.Printf("%s attempt %d: error making request: %v", wh.Message, wh.Attempts+1, err)
		return false
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("%s attempt %d: bad status %d", wh.Message, wh.Attempts+1, resp.StatusCode)
		return false
	}
	return true
}

const maxAttempts = 5

func handle(client *http.Client, wh queue.WebHook) {

	if deliverOnce(client, wh) {
		recordDone(wh)
		return
	}

	wh.Attempts++

	if wh.Attempts >= maxAttempts {
		recordDead(wh)
		return
	}

	delay := min(time.Minute<<(wh.Attempts-1), time.Hour)
	delay += rand.N(delay / 10)
	time.AfterFunc(delay, func() { queue.WebhookQueue <- wh })

}

func Scheduler(worker int) {

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for range worker {
		go func() {
			for wh := range queue.WebhookQueue {
				handle(client, wh)
			}
		}()
	}
}
