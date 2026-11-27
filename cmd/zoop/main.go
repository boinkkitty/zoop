// Command zoop is the whole app: the JSON API and the built frontend, from one origin.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata" // signup validates IANA zones; don't depend on the image having them

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/boinkkitty/zoop/internal/app"
	"github.com/boinkkitty/zoop/internal/clock"
	"github.com/boinkkitty/zoop/internal/db"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("zoop exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	srv := &http.Server{
		Addr: ":" + port,
		Handler: app.New(app.Deps{
			DB:     pool,
			Clock:  clock.System{},
			Mailer: mail.Unconfigured{},
			Assets: web.Dist(),
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("zoop listening", "port", port)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
