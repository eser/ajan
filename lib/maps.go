package lib

import "strings"

func CaseInsensitiveSet(m *map[string]any, key string, value any) {
	for k := range *m {
		if strings.EqualFold(k, key) {
			(*m)[k] = value

			return
		}
	}

	(*m)[key] = value
}
