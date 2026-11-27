// Command zoop is the whole app: the JSON API and the built frontend, from one origin.
// It never changes the schema: apply internal/db/migrations by hand with psql first.
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
	"github.com/boinkkitty/zoop/internal/google"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/web"
)

// sweepEvery is how often expired sessions and rate-limit rows are deleted.
const sweepEvery = 15 * time.Minute

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

	googleCfg, err := googleConfig()
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	c, shift := withTestClock(clock.System{})
	deps := app.Deps{
		DB:     pool,
		Clock:  c,
		Mailer: mail.Unconfigured{},
		Assets: web.Dist(),
		Google: googleCfg,
	}
	go app.KeepSwept(ctx, deps, sweepEvery)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           shift(app.New(deps)),
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

// googleConfig reads Google sign-in's OAuth client from the environment. With no
// GOOGLE_CLIENT_ID, Google sign-in is off and the button sends people back to sign in.
func googleConfig() (*google.Config, error) {
	id := os.Getenv("GOOGLE_CLIENT_ID")
	if id == "" {
		slog.Warn("GOOGLE_CLIENT_ID is not set: Google sign-in is off")
		return nil, nil
	}
	cfg := &google.Config{
		Issuer:       google.Issuer,
		ClientID:     id,
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"), // https://<host>/api/v1/auth/google/callback
	}
	if cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID is set, so GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URL must be too")
	}
	return cfg, nil
}
