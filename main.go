package main

import (
	"fmt"
	"net/http"
	"webhook/job"
	"webhook/queue"
)

func callBackCheck(http.ResponseWriter, *http.Request) {
	fmt.Print("hitting the api ")
}

func main() {

	job.Scheduler(10)

	handler := http.NewServeMux()

	handler.HandleFunc("/hello", queue.PushTOQueueAndSendResponse)
	handler.HandleFunc("/callback-check", callBackCheck)
	handler.HandleFunc("/create-req", job.CreateSecret)

	fmt.Println("Server running on port 8000")
	err := http.ListenAndServe(":8000", handler)

	if err != nil {
		fmt.Println(err)
	}

}
