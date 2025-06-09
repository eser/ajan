package lib

import (
	"log/slog"
	"strings"
)

func SerializeSlogAttrs(attrs []slog.Attr) string {
	var b strings.Builder

	for i, attr := range attrs {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(attr.String())
	}

	return b.String()
}
