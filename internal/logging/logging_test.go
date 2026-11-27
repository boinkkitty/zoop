package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/boinkkitty/zoop/internal/logging"
)

// A plain string logged under a key that isn't on the allowlist never reaches the logs.
func TestLoggerDropsFieldsNotOnTheAllowlist(t *testing.T) {
	var buf bytes.Buffer
	log := logging.New(&buf)

	log.Info("request",
		"status", 200,
		"email", "ada@example.com",
		"query", "dear diary",
		slog.Group("req", "body", `{"text":"dear diary"}`),
	)

	if strings.Contains(buf.String(), "ada@example.com") || strings.Contains(buf.String(), "diary") {
		t.Errorf("log line leaks a field off the allowlist: %s", buf.String())
	}
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line %q is not JSON: %v", buf.String(), err)
	}
	if line["status"] != float64(200) || line["msg"] != "request" {
		t.Errorf("log line = %v, want allowlisted status and msg kept", line)
	}
}
