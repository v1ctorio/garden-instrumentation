package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

var SLACK_LOG_WEBHOOK_URL = os.Getenv("SLACK_LOG_WEBHOOK_URL")

func LoadAPIKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	raw := os.Getenv("INGEST_API_KEYS")

	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = struct{}{}
		}
	}

	return keys
}

type slackWebhookBody struct {
	Text string `json:"text"`
}

func LogToSlack(text string) {
	if SLACK_LOG_WEBHOOK_URL == "" {
		return
	}
	marshalled, err := json.Marshal(slackWebhookBody{
		Text: text,
	})
	if err != nil {
		fmt.Print("Unable to marshal text in LogToSlack")
		return
	}

	http.Post(SLACK_LOG_WEBHOOK_URL, "application/json", bytes.NewReader(marshalled))
}
