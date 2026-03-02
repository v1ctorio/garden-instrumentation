package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/slack-go/slack"

	"ingestion-orchid/config"
	. "ingestion-orchid/internal"
)

func main() {

	ctx := context.Background()

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
		}),
	))

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	slack_api := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	apiKeys := LoadAPIKeys()

	r := chi.NewRouter()

	r.Get("/health", healthcheckHandler(db))
	r.Post("/slack/events", SlackEventsHandler(db, slack_api, os.Getenv("SLACK_SIGNING_SECRET")))
	r.Get("/recognized-events", advertiseRecognizedEvents)

	r.Route("/instrumentation", func(r chi.Router) {
		r.Use(ApiKeyAuth(apiKeys))
		r.Post("/user", userHandler(db))
		r.Post("/event", eventHandler(db))

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

		slog.Info("Received request to record user", "user", u)

		if u.SlackID == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
			return
		}

		if u.JoinTimestamp == nil {
			// Replace with new go 1.26 fancy feature new()
			now := time.Now().UTC()
			u.JoinTimestamp = &now
		}

		if err := InsertUser(r.Context(), db, u); err != nil {
			http.Error(w, "failed to store user", http.StatusInternalServerError)
			slog.Error("Failed to record user", "error", err)
			return
		}

		LogToSlack("New user recorded")
		w.WriteHeader(http.StatusAccepted)
	}
}

func eventHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var e Event

		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		slog.Info("Received request to record event", "event", e)

		if e.SlackID == "" || e.EventKind == "" {
			http.Error(w, "malformed request", http.StatusBadRequest)
			return
		}

		if e.Metadata == nil {
			e.Metadata = map[string]string{}
		}

		if err := RecordEvent(r.Context(), db, e); err != nil {
			http.Error(w, "failed to store event", http.StatusInternalServerError)
			slog.Error("Failed to record event", "error", err)
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

func advertiseRecognizedEvents(w http.ResponseWriter, r *http.Request) {

	events := config.RecognizedEvents

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
