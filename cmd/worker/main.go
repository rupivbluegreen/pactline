package main

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	hostPort := envOr("TEMPORAL_HOST", "localhost:7233")
	namespace := envOr("TEMPORAL_NAMESPACE", "default")
	taskQueue := envOr("PACTLINE_TASK_QUEUE", "pactline")

	c, err := client.Dial(client.Options{HostPort: hostPort, Namespace: namespace})
	if err != nil {
		slog.Error("temporal dial", "err", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, taskQueue, worker.Options{})

	slog.Info("worker starting", "task_queue", taskQueue, "host", hostPort)
	if err := w.Run(worker.InterruptCh()); err != nil {
		slog.Error("worker run", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
