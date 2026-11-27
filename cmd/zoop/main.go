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
	"github.com/boinkkitty/zoop/internal/errreport"
	"github.com/boinkkitty/zoop/internal/google"
	"github.com/boinkkitty/zoop/internal/logging"
	"github.com/boinkkitty/zoop/internal/mail"
	"github.com/boinkkitty/zoop/internal/mail/resend"
	"github.com/boinkkitty/zoop/web"
)

// sweepEvery is how often expired sessions and rate-limit rows are deleted.
const sweepEvery = 15 * time.Minute

func main() {
	slog.SetDefault(logging.New(os.Stdout))
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

	// Error reporting is off unless SENTRY_DSN is set.
	reports, err := errreport.NewClient(errreport.Options{
		DSN:         os.Getenv("SENTRY_DSN"),
		Release:     os.Getenv("SENTRY_RELEASE"),
		Environment: os.Getenv("SENTRY_ENVIRONMENT"),
	})
	if err != nil {
		return fmt.Errorf("set up Sentry: %w", err)
	}
	defer reports.Flush(2 * time.Second)

	googleCfg, err := googleConfig()
	if err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	mailer, appURL, err := mailConfig(port)
	if err != nil {
		return err
	}

	c, shift := withTestClock(clock.System{})
	mailer, outbox := withTestOutbox(mailer)
	deps := app.Deps{
		DB:     pool,
		Clock:  c,
		Mailer: mailer,
		AppURL: appURL,
		Assets: web.Dist(),
		Google: googleCfg,
		Logs:   os.Stdout,
		Sentry: reports,
	}
	go app.KeepSwept(ctx, deps, sweepEvery)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           shift(outbox(app.New(deps))),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("zoop listening on :" + port)
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

// mailConfig reads the Resend account and the app's public URL, which emailed links point at.
// With no RESEND_API_KEY, no email is sent: signup still works, and the failures are reported.
func mailConfig(port string) (mail.Mailer, string, error) {
	appURL := os.Getenv("APP_URL") // https://<host>
	key := os.Getenv("RESEND_API_KEY")
	if key == "" {
		slog.Warn("RESEND_API_KEY is not set: no email will be sent")
		if appURL == "" {
			appURL = "http://localhost:" + port
		}
		return mail.Unconfigured{}, appURL, nil
	}
	from := os.Getenv("MAIL_FROM") // e.g. Zoop <hello@zoop.app>, on a domain verified in Resend
	if from == "" || appURL == "" {
		return nil, "", errors.New("RESEND_API_KEY is set, so MAIL_FROM and APP_URL must be too")
	}
	return resend.New(resend.Config{APIKey: key, From: from}), appURL, nil
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
