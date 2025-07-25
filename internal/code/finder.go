package code

import (
	"github.com/a-peyrard/mm/internal/set"
	"io/fs"
	"path/filepath"
)

type Consumer[T any] func(T) error

// fixme: find a better place for this
var dirToSkip = set.Of(".venv", ".git", "node_modules", "venv", "__pycache__", ".idea", ".vscode")

func FindInDirectory(dir string, extensions set.Set[string], callback Consumer[string]) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && dirToSkip.Contains(d.Name()) {
			return fs.SkipDir
		}
		if !d.IsDir() && extensions.Contains(filepath.Ext(d.Name())) {
			err := callback(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
