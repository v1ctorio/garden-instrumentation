package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/slack-go/slack"

	. "ingestion-orchid/internal"
)

func main() {

	ctx := context.Background()

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	slack_api := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	apiKeys := LoadAPIKeys()
	allowedEvents, err := LoadAllowedEvents(ctx, db)

	r := chi.NewRouter()

	r.Get("/health", healthcheckHandler(db))
	r.Post("/slack/events", SlackEventsHandler(db, slack_api, os.Getenv("SLACK_SIGNING_SECRET")))

	r.Route("/instrumentation", func(r chi.Router) {
		r.Use(ApiKeyAuth(apiKeys))
		r.Post("/user", userHandler(db))
		r.Post("/event", eventHandler(db, allowedEvents))

	})

	fmt.Println("Hello chat, listening on :8400")

	http.ListenAndServe(":8400", r)
}

func userHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User

		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if u.SlackID == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
			return
		}

		if err := InsertUser(r.Context(), db, u); err != nil {
			http.Error(w, "failed to store user", http.StatusInternalServerError)
			return
		}

		LogToSlack("New user recorded")
		w.WriteHeader(http.StatusAccepted)
	}
}

func eventHandler(db *pgxpool.Pool, allowed map[string]struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e Event

		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if _, ok := allowed[e.EventName]; !ok {
			http.Error(w, "unknwon event_name", http.StatusBadRequest)
		}

		if e.SlackID == "" || e.EventName == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
		}

		if e.Metadata == nil {
			e.Metadata = map[string]string{}
		}

		if err := RecordEvent(r.Context(), db, e); err != nil {
			http.Error(w, "failed to store event", http.StatusInternalServerError)
			return
		}

		LogToSlack("New event recorded")
		w.WriteHeader(http.StatusAccepted)
	}
}

func healthcheckHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.Ping(r.Context())

		if err != nil {
			http.Error(w, "bad", http.StatusInternalServerError)
			fmt.Printf("Healthcheck failed: %v\n", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
