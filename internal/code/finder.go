package code

import (
	"github.com/a-peyrard/mm/internal/set"
	"os"
	"path/filepath"
)

type Consumer[T any] func(T) error

func FindInDirectory(dir string, extensions set.Set[string], callback Consumer[string]) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && extensions.Contains(filepath.Ext(info.Name())) {
			err := callback(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
