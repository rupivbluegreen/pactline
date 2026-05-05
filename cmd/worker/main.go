package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/rupivbluegreen/pactline/internal/ai"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/document"
	"github.com/rupivbluegreen/pactline/internal/storage"
	pactlineworkflow "github.com/rupivbluegreen/pactline/internal/workflow"
	"github.com/rupivbluegreen/pactline/internal/workflow/activities"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	hostPort := envOr("TEMPORAL_HOST", "localhost:7233")
	namespace := envOr("TEMPORAL_NAMESPACE", "default")
	taskQueue := envOr("PACTLINE_TASK_QUEUE", pactlineworkflow.WorkflowTaskQueue)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	st, err := storage.New(ctx, storage.ConfigFromEnv())
	if err != nil {
		slog.Error("storage init", "err", err)
		os.Exit(1)
	}

	docCli, err := document.Dial(ctx, envOr("DOCUMENT_SIDECAR_ADDR", "localhost:50052"))
	if err != nil {
		slog.Error("document dial", "err", err)
		os.Exit(1)
	}
	defer docCli.Close()

	aiCli, err := ai.Dial(ctx, envOr("AI_SIDECAR_ADDR", "localhost:50051"))
	if err != nil {
		slog.Error("ai dial", "err", err)
		os.Exit(1)
	}
	defer aiCli.Close()

	c, err := client.Dial(client.Options{HostPort: hostPort, Namespace: namespace})
	if err != nil {
		slog.Error("temporal dial", "err", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflow(pactlineworkflow.ContractLifecycleWorkflow)

	acts := activities.New(
		pool, st, aiCli, docCli,
		envOr("PACTLINE_AI_MODEL", "anthropic/claude-3-5-sonnet"),
		envOr("PACTLINE_AI_PROMPT_VERSION", "v1"),
	)
	w.RegisterActivity(acts.TransitionContract)
	w.RegisterActivity(acts.ParseDocument)
	w.RegisterActivity(acts.PersistParsedText)
	w.RegisterActivity(acts.ExtractFields)
	w.RegisterActivity(acts.PersistExtractedFields)

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
