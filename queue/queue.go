package queue

import (
	"encoding/json"
	"io"
	"net/http"
	"webhook/secret"
)

type ResponseData struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type WebHook struct {
	CallBackUrl string
	Message     string
}

type WebHookRequest struct {
	RequestID   string `json:"request_id"`
	Message     string `json:"message"`
	CallbackURL string `json:"callbackUrl"`
}

var WebhookQueue = make(chan WebHook)

func PushTOQueueAndSendResponse(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("X-Signature")
	if !secret.VerifyHMACSHA256(string(body), sig) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var req WebHookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	WebhookQueue <- WebHook{
		CallBackUrl: req.CallbackURL,
		Message:     req.Message,
	}

	w.Header().Set("Content-Type", "application/json")

	d := ResponseData{
		Status:  "success",
		Message: "Hello, World!",
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
