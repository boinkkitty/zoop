// Package pages holds the rules for Pages: one Notebook's writing for one Day.
package pages

import "log/slog"

// Content is a Page's document as stored (TipTap JSON, ADR 0001). It's the
// most private thing Zoop holds, so it never appears in logs.
type Content string

func (Content) LogValue() slog.Value { return slog.StringValue("[redacted]") }
