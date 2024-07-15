package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type config struct {
	port      int
	redisAddr string
}

type application struct {
	config config
	logger *slog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 9900, "api server port")
	flag.StringVar(&cfg.redisAddr, "redisAddr", "localhost:6379", "redis server address")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := application{
		config: cfg,
		logger: logger,
	}

	client := redis.NewClient(&redis.Options{
		Addr: "100.24.18.53:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("could not connect to redis",
			slog.String("error", err.Error()),
		)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/redis-status/{task_id}", app.fetchRedisStatus(client))
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.port),
		Handler: mux,
	}

	logger.Info("starting server", slog.Int("port", cfg.port))
	err := srv.ListenAndServe()
	if err != nil {
		logger.Error("server", slog.String("error", err.Error()))
	}
}

func (app *application) fetchRedisStatus(client *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task_id := r.PathValue("task_id")
		if _, err := uuid.Parse(task_id); err != nil {
			app.writeResponse(w, "Invalid task id", http.StatusBadRequest, nil, err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		status, err := client.HGetAll(ctx, task_id).Result()
		if err != nil {
			app.logger.Error("fetching task", slog.String("error", err.Error()))
			app.writeResponse(w, "Status fetch failed", http.StatusInternalServerError, nil, err)
			return
		}
		app.writeResponse(w, "Successful", http.StatusOK, status, nil)
	}
}

func (app *application) writeResponse(w http.ResponseWriter, message string, status int, data any, err error) {
	resp := map[string]any{
		"message": message,
		"status":  status,
		"data":    data,
		"error":   err,
	}
	js, err := json.Marshal(resp)
	if err != nil {
		app.logger.Error("writing json", slog.String("error", err.Error()))
		w.WriteHeader(500)
		return
	}
	js = append(js, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
}
