package secrets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Argument represent a key/value pair that was passed in as an environment
// variable. In a linux system, it would be represented as a `KEY=Value` environment variable.
// The argument list is stored as a JSON payload, and those values needs to be extracted
// from it before it can be passed to buildkit.
type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func ReadKeyValueFromDir(ctx context.Context, path string) (collection []KeyValue, err error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			return nil, fmt.Errorf("E#1014: unexpected directory in path, only text files should be present")
		}

		data, err := os.ReadFile(filepath.Join(path, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("E#1014: error while reading the content of the file (%s) -> %w", filepath.Join(path, e.Name()), err)
		}

		collection = append(collection, KeyValue{
			Key:   e.Name(),
			Value: string(data),
		})
	}

	return collection, nil

}
