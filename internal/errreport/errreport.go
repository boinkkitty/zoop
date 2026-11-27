// Package errreport is the Sentry adapter. Its client sends only what's on an
// allowlist: the error and its stack, where it happened, and the user's id. It
// never sends request data, breadcrumbs, emails or anyone's writing.
package errreport

import (
	"github.com/getsentry/sentry-go"
)

// Options configure the client. Production leaves Transport nil and sets DSN
// from SENTRY_DSN; an empty DSN turns reporting off.
type Options struct {
	DSN         string
	Release     string
	Environment string
	Transport   sentry.Transport // tests record events here instead of sending them
}

// NewClient returns a Sentry client whose every event passes through scrub.
func NewClient(o Options) (*sentry.Client, error) {
	return sentry.NewClient(sentry.ClientOptions{
		Dsn:              o.DSN,
		Release:          o.Release,
		Environment:      o.Environment,
		Transport:        o.Transport,
		SendDefaultPII:   false,
		AttachStacktrace: true,
		MaxBreadcrumbs:   -1, // none at all
		BeforeBreadcrumb: func(*sentry.Breadcrumb, *sentry.BreadcrumbHint) *sentry.Breadcrumb { return nil },
		BeforeSend: func(e *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			return scrub(e)
		},
	})
}

// Tags and contexts an event may keep. Everything else is dropped.
var (
	allowedTags     = map[string]bool{"route": true, "method": true, "request_id": true}
	allowedContexts = map[string]bool{"runtime": true, "os": true, "device": true, "trace": true}
)

// scrub rebuilds e from allowlisted fields only, so anything a future
// integration attaches (request, extra, message, modules) never leaves.
func scrub(e *sentry.Event) *sentry.Event {
	out := &sentry.Event{
		EventID:     e.EventID,
		Level:       e.Level,
		Platform:    e.Platform,
		Release:     e.Release,
		Dist:        e.Dist,
		Environment: e.Environment,
		Sdk:         e.Sdk,
		Timestamp:   e.Timestamp,
		Fingerprint: e.Fingerprint,
		Exception:   e.Exception,
		Threads:     e.Threads,
		User:        sentry.User{ID: e.User.ID},
		Tags:        map[string]string{},
		Contexts:    map[string]sentry.Context{},
	}
	for k, v := range e.Tags {
		if allowedTags[k] {
			out.Tags[k] = v
		}
	}
	for k, v := range e.Contexts {
		if allowedContexts[k] {
			out.Contexts[k] = v
		}
	}
	return out
}
