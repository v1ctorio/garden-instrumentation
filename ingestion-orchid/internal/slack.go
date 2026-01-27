package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func SlackEventsHandler(db *pgxpool.Pool, api *slack.Client, signingSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("Failed to decode slack events req")
			return
		}
		sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println("Received slack event:", eventsAPIEvent.Type)
		switch eventsAPIEvent.Type {
		case slackevents.URLVerification:
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		case slackevents.CallbackEvent:
			w.WriteHeader(http.StatusOK)
			innerEvent := eventsAPIEvent.InnerEvent
			fmt.Println("Received inner slack event:", innerEvent.Type)
			switch ev := innerEvent.Data.(type) {
			case *slackevents.UserChangeEvent:
				if ev.User.IsRestricted {
					return
				}
				uIsRestricted := false
				err := db.QueryRow(r.Context(), `
				SELECT is_restricted
				FROM users
				WHERE slack_id=$1
				`, ev.User.ID).Scan(uIsRestricted)
				if err != nil {
					log.Printf("Failed to query updated user %v", err)
					return
				}
				if uIsRestricted == false {
					return
				}

				_, err = db.Exec(r.Context(), `
				UPDATE users
				SET timezone=$1, is_restricted=$2
				WHERE slack_id=$3
				`, ev.User.TZ, ev.User.IsRestricted, ev.User.ID)
			}
		}

	}
}
