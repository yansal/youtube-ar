package logfmt

import (
	"fmt"
	"sort"
	"strings"
)

// Marshal marshals v.
func Marshal(m map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var fields []string
	for _, key := range keys {
		format := "%s=%s"
		value := m[key]
		if needquote(value) {
			format = "%s=%q"
		}
		fields = append(fields, fmt.Sprintf(format, key, value))
	}
	return []byte(strings.Join(fields, " ")), nil
}

func needquote(s string) bool {
	return strings.Contains(s, " ")
}
