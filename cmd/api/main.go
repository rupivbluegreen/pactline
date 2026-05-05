package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.temporal.io/sdk/client"

	"github.com/rupivbluegreen/pactline/internal/api"
	"github.com/rupivbluegreen/pactline/internal/api/handlers"
	"github.com/rupivbluegreen/pactline/internal/auth"
	"github.com/rupivbluegreen/pactline/internal/database"
	"github.com/rupivbluegreen/pactline/internal/email"
	"github.com/rupivbluegreen/pactline/internal/organizations"
	"github.com/rupivbluegreen/pactline/internal/storage"
	pactlineworkflow "github.com/rupivbluegreen/pactline/internal/workflow"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := envOr("PACTLINE_API_ADDR", ":8000")
	baseURL := envOr("PACTLINE_BASE_URL", "http://localhost:3000")
	tqueue := envOr("PACTLINE_TASK_QUEUE", pactlineworkflow.WorkflowTaskQueue)
	tHost := envOr("TEMPORAL_HOST", "localhost:7233")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, database.DSNFromEnv())
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.MigrateUp(ctx, pool); err != nil {
		slog.Error("migrate up", "err", err)
		os.Exit(1)
	}

	st, err := storage.New(ctx, storage.ConfigFromEnv())
	if err != nil {
		slog.Error("storage init", "err", err)
		os.Exit(1)
	}

	tcli, err := client.Dial(client.Options{HostPort: tHost})
	if err != nil {
		slog.Error("temporal dial", "err", err)
		os.Exit(1)
	}
	defer tcli.Close()

	users := database.NewUserRepo(pool)
	orgs := database.NewOrganizationRepo(pool)
	mems := database.NewMembershipRepo(pool)
	sess := database.NewSessionRepo(pool)
	links := database.NewMagicLinkRepo(pool)
	mails := email.NewSender()

	authSvc := auth.NewService(pool, users, links, sess, mails, baseURL)
	orgSvc := organizations.NewService(pool)

	deps := api.Deps{
		Sessions:    sess,
		Memberships: mems,
		Auth:        &handlers.AuthHandlers{Svc: authSvc},
		Me:          &handlers.MeHandler{Users: users, Memberships: mems, Orgs: orgs},
		Orgs:        &handlers.OrgsHandlers{Svc: orgSvc},
		Contracts: &handlers.ContractsHandler{
			Pool:              pool,
			Storage:           st,
			Contracts:         database.NewContractRepo(pool),
			ContractTypes:     database.NewContractTypeRepo(pool),
			ContractDocuments: database.NewContractDocumentRepo(pool),
			ExtractedFields:   database.NewExtractedFieldRepo(pool),
			Temporal:          tcli,
			TaskQueue:         tqueue,
		},
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.Router(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("api listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("api crashed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("api shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("api shutdown failed", "err", err)
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
