package instance

import "dirschema/internal/fswalk"

const DefaultMaxContentBytes int64 = 1 << 20

func ScanAttributes(schema map[string]any) fswalk.Options {
	opts := fswalk.Options{MaxContentBytes: DefaultMaxContentBytes, SymlinkPolicy: fswalk.SymlinkRecord}
	scanMap(schema, &opts)
	return opts
}

func scanMap(node map[string]any, opts *fswalk.Options) {
	for key, value := range node {
		switch key {
		case "properties":
			props, ok := value.(map[string]any)
			if ok {
				for propKey := range props {
					switch propKey {
					case "size":
						opts.IncludeSize = true
					case "sha256":
						opts.IncludeSHA256 = true
					case "content":
						opts.IncludeContent = true
					}
				}
				for _, child := range props {
					if childMap, ok := child.(map[string]any); ok {
						scanMap(childMap, opts)
					}
				}
			}
		default:
			if childMap, ok := value.(map[string]any); ok {
				scanMap(childMap, opts)
			}
		}
	}
}
