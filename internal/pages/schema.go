package pages

import (
	"encoding/json"
	"fmt"
)

// The v1 Page schema: which nodes and marks a Page may hold and where.
// web/src/page/schema.ts declares the same grammar for the editor, and the
// shared fixtures in testdata/page-docs keep the two in step.

// MaxListDepth is how deeply lists may nest: a list inside at most two others.
const MaxListDepth = 3

// markTypes are the v1 marks. None has attrs (highlight has one colour).
var markTypes = set("bold", "italic", "underline", "highlight")

// blocks is ProseMirror's "block" group: what a doc, a quote or a list item may hold.
var blocks = set("paragraph", "bulletList", "orderedList", "taskList", "blockquote")

// lists are the nodes that count towards MaxListDepth.
var lists = set("bulletList", "orderedList", "taskList")

// contentRule is a container node's content expression: an optional required
// first child, then any number of children from each (at least one if nonEmpty).
type contentRule struct {
	first    string
	each     map[string]bool
	nonEmpty bool
}

// containers are the nodes that hold blocks. The paragraph (text only) and the
// doc (per Notebook, see docRule) are handled apart.
var containers = map[string]contentRule{
	"blockquote":  {each: blocks, nonEmpty: true},
	"bulletList":  {each: set("listItem"), nonEmpty: true},
	"orderedList": {each: set("listItem"), nonEmpty: true},
	"taskList":    {each: set("taskItem"), nonEmpty: true},
	"listItem":    {first: "paragraph", each: blocks},
	"taskItem":    {first: "paragraph", each: blocks},
}

// docRule is a Notebook's top-level grammar. My Journal's is `paragraph block*`:
// the first paragraph is the Title, so it can never be a list or a quote.
func docRule(notebook string) contentRule {
	return contentRule{first: "paragraph", each: blocks}
}

// checkAttrs checks a node's attrs against what the editor writes for its type.
// Only a task item has attrs: {"checked": bool}, always present.
func checkAttrs(typ string, raw json.RawMessage) error {
	if typ != "taskItem" {
		if !noAttrs(raw) {
			return fmt.Errorf("%s has no attrs", typ)
		}
		return nil
	}
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil || len(attrs) != 1 {
		return fmt.Errorf(`a taskItem's attrs are exactly {"checked": true|false}`)
	}
	var checked bool
	if c, ok := attrs["checked"]; !ok || json.Unmarshal(c, &checked) != nil || string(c) == "null" {
		return fmt.Errorf(`a taskItem's attrs are exactly {"checked": true|false}`)
	}
	return nil
}

func (r contentRule) check(children []node) error {
	if r.nonEmpty && len(children) == 0 {
		return fmt.Errorf("needs at least one child")
	}
	for i, c := range children {
		if i == 0 && r.first != "" {
			if c.Type != r.first {
				return fmt.Errorf("must start with a %s, not %q", r.first, c.Type)
			}
			continue
		}
		if !r.each[c.Type] {
			return fmt.Errorf("%q can't sit here", c.Type)
		}
	}
	if r.first != "" && len(children) == 0 {
		return fmt.Errorf("must start with a %s", r.first)
	}
	return nil
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}
