package expand

import (
	"sort"
	"strings"
)

// FormatListDSL converts a fswalk-style instance into list-form DSL.
func FormatListDSL(instance map[string]any) []any {
	return dirList(instance)
}

func dirList(node map[string]any) []any {
	keys := make([]string, 0, len(node))
	for key := range node {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	entries := make([]any, 0, len(keys))
	for _, key := range keys {
		value := node[key]
		if strings.HasSuffix(key, "/") {
			child, _ := value.(map[string]any)
			entries = append(entries, map[string]any{key: dirList(child)})
			continue
		}
		switch v := value.(type) {
		case bool:
			if v {
				entries = append(entries, key)
			}
		case map[string]any:
			entries = append(entries, map[string]any{key: v})
		}
	}
	return entries
}
