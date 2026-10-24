package queue

import (
	"encoding/json"
	"io"
	"net/http"
	"uuid"
	"webhook/secret"
)

type ResponseData struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Id      uuid.UUID `json:"id"`
}

type WebHook struct {
	id           uuid.UUID
	CallBackUrl  string
	Message      string
	requestBody  string
	responseBody string
	status       Status
	Attempts     int
}

type Status int

const (
	Pending Status = iota
	Active
	Inactive
	Deleted
)

type WebHookRequest struct {
	RequestID   string `json:"request_id"`
	Message     string `json:"message"`
	CallbackURL string `json:"callbackUrl"`
}

var WebhookQueue = make(chan WebHook, 1000)

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

	uId := uuid.New()

	w.Header().Set("Content-Type", "application/json")

	select {
	case WebhookQueue <- WebHook{
		CallBackUrl: req.CallbackURL,
		Message:     req.Message,
		id:          uId,
	}:
		w.WriteHeader(http.StatusAccepted)

	default:
		http.Error(w, "queue full please wait", http.StatusServiceUnavailable)
		return
	}

	d := ResponseData{
		Status:  "accepted",
		Message: "webhook queued",
		Id:      uId,
	}

	if err := json.NewEncoder(w).Encode(d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
