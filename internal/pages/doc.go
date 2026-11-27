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
const SchemaVersion = 1

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

// grammar is the shape of one Notebook's document.
type grammar struct {
	// titled: the first block is the Day's Title, always a plain paragraph (`paragraph block*`).
	// Otherwise any blocks will do (`block+`).
	titled bool
}

// grammars are the v1 Notebooks' documents, by slug. The client's (web/src/page/schema.ts) match.
var grammars = map[string]grammar{
	"my-journal": {titled: true},
	"i-learned":  {titled: false},
}

// ParseDoc checks that doc is a Page for notebook in the current schema and returns its plain text:
// formatting removed, one line per block.
//
// A block so far is a paragraph holding text. My Journal is `paragraph block*`: its first paragraph
// is the Title, whose bold is CSS, never a stored mark. I Learned is `block+`, with no Title.
func ParseDoc(notebook string, doc json.RawMessage) (string, error) {
	g, ok := grammars[notebook]
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
		if g.titled {
			return "", invalid("a Page starts with its Title paragraph")
		}
		return "", invalid("a Page has at least one block")
	}
	lines := make([]string, 0, len(*root.Content))
	for i, block := range *root.Content {
		if i == 0 && g.titled && block.Type != "paragraph" {
			return "", invalid("the Title is a plain paragraph, not %q", block.Type)
		}
		line, err := paragraph(block)
		if err != nil {
			return "", invalid("block %d: %v", i, err)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

func paragraph(n node) (string, error) {
	if n.Type != "paragraph" {
		return "", fmt.Errorf("%q is not a block in this schema", n.Type)
	}
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
		if len(inline.Marks) > 0 && string(inline.Marks) != "null" {
			return "", fmt.Errorf("no marks are in this schema yet")
		}
		b.WriteString(*inline.Text)
	}
	return b.String(), nil
}

func noAttrs(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == "{}"
}
