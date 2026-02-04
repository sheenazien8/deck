package ui

import (
	"context"
	"sort"
	"strings"

	"github.com/a-h/templ"
)

// spread converts a map[string]string to a string of HTML attributes
// in key-sorted order. It's used by generated templ code.
func spread(attrs map[string]string) string {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(templ.EscapeString(k))
		b.WriteString(`="`)
		b.WriteString(templ.EscapeString(attrs[k]))
		b.WriteString(`"`)
	}
	return b.String()
}

// renderToString renders a templ.Component to a string using the provided context.
func renderToString(ctx context.Context, c templ.Component) (string, error) {
	if c == nil {
		return "", nil
	}
	b := templ.GetBuffer()
	defer templ.ReleaseBuffer(b)
	if err := c.Render(ctx, b); err != nil {
		return "", err
	}
	return b.String(), nil
}
