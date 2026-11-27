package pages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaVersion is the version of the Page document format this server reads and writes.
// The client's constant (web/src/page/schema.ts) must match it, and so must
// testdata/page-docs/schema-version.json.
//
// 2: formatting (marks, lists, checklists, quotes).
const SchemaVersion = 2

// EmptyDoc is an unwritten Page: one empty paragraph, the Title line.
var EmptyDoc = json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)

// InvalidDocError means a saved doc isn't a Page in the current schema.
type InvalidDocError struct{ Reason string }

func (e *InvalidDocError) Error() string { return "pages: invalid doc: " + e.Reason }

func invalid(format string, a ...any) error { return &InvalidDocError{fmt.Sprintf(format, a...)} }

// node is a ProseMirror node as TipTap's getJSON writes it.
type node struct {
	Type    string          `json:"type"`
	Attrs   json.RawMessage `json:"attrs"`
	Content *[]node         `json:"content"`
	Text    *string         `json:"text"`
	Marks   json.RawMessage `json:"marks"`
}

// ParseDoc checks that doc is a Page for notebook in the current schema (schema.go) and returns
// its plain text: formatting, list markers and checkboxes dropped, one line per paragraph, and so
// one line per block wherever it sits (in a list, a checklist or a quote).
//
// In My Journal the first paragraph is the Title; its bold is CSS, never a stored mark.
// I Learned has no Title.
func ParseDoc(notebook string, doc json.RawMessage) (string, error) {
	rule, ok := docRules[notebook]
	if !ok {
		return "", invalid("there's no Notebook %q", notebook)
	}
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.DisallowUnknownFields()
	var root node
	if err := dec.Decode(&root); err != nil {
		return "", invalid("not a document: %v", err)
	}
	if root.Type != "doc" || root.Text != nil || len(root.Marks) > 0 || !noAttrs(root.Attrs) {
		return "", invalid("the root must be a plain doc node")
	}
	if root.Content == nil || len(*root.Content) == 0 {
		if rule.first != "" {
			return "", invalid("a Page starts with its Title paragraph")
		}
		return "", invalid("a Page has at least one block")
	}
	if err := rule.check(*root.Content); err != nil {
		return "", invalid("doc: %v", err)
	}
	var lines []string
	for i, block := range *root.Content {
		if err := walk(block, 0, &lines); err != nil {
			return "", invalid("block %d: %v", i, err)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// walk checks a block node and its descendants, appending one line per paragraph.
// depth is how many lists the node already sits inside.
func walk(n node, depth int, lines *[]string) error {
	if n.Type == "paragraph" {
		line, err := paragraph(n)
		if err != nil {
			return err
		}
		*lines = append(*lines, line)
		return nil
	}
	rule, ok := containers[n.Type]
	if !ok {
		return fmt.Errorf("%q is not a block in this schema", n.Type)
	}
	if n.Text != nil || len(n.Marks) > 0 {
		return fmt.Errorf("a %s has no text or marks of its own", n.Type)
	}
	if err := checkAttrs(n.Type, n.Attrs); err != nil {
		return err
	}
	if lists[n.Type] {
		depth++
		if depth > MaxListDepth {
			return fmt.Errorf("lists nest at most %d levels", MaxListDepth)
		}
	}
	var children []node
	if n.Content != nil {
		children = *n.Content
	}
	if err := rule.check(children); err != nil {
		return fmt.Errorf("%s: %v", n.Type, err)
	}
	for _, c := range children {
		if err := walk(c, depth, lines); err != nil {
			return err
		}
	}
	return nil
}

func paragraph(n node) (string, error) {
	if n.Text != nil || len(n.Marks) > 0 || !noAttrs(n.Attrs) {
		return "", fmt.Errorf("a paragraph has no text, marks or attrs of its own")
	}
	if n.Content == nil {
		return "", nil
	}
	var b strings.Builder
	for _, inline := range *n.Content {
		if inline.Type != "text" {
			return "", fmt.Errorf("%q can't sit inside a paragraph", inline.Type)
		}
		if inline.Text == nil || *inline.Text == "" {
			return "", fmt.Errorf("a text node needs some text")
		}
		if inline.Content != nil || !noAttrs(inline.Attrs) {
			return "", fmt.Errorf("a text node is a leaf with no attrs")
		}
		if err := checkMarks(inline.Marks); err != nil {
			return "", err
		}
		b.WriteString(*inline.Text)
	}
	return b.String(), nil
}

// checkMarks checks a text node's marks: absent, or a non-empty list of distinct v1 marks.
func checkMarks(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var marks []struct {
		Type  string          `json:"type"`
		Attrs json.RawMessage `json:"attrs"`
	}
	if err := dec.Decode(&marks); err != nil {
		return fmt.Errorf("marks: %v", err)
	}
	if len(marks) == 0 {
		return fmt.Errorf("text with no marks has no marks list")
	}
	seen := map[string]bool{}
	for _, m := range marks {
		if !markTypes[m.Type] {
			return fmt.Errorf("%q is not a mark in this schema", m.Type)
		}
		if seen[m.Type] {
			return fmt.Errorf("%q appears twice", m.Type)
		}
		seen[m.Type] = true
		if !noAttrs(m.Attrs) {
			return fmt.Errorf("a %s mark has no attrs", m.Type)
		}
	}
	return nil
}

func noAttrs(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == "{}"
}
