package embedding

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed python/indexer.py
var pythonScript []byte

//go:embed python/pyproject.toml
var pyprojectToml []byte

type (
	IndexerOptions struct {
		DbPath string
	}

	IndexerOption func(*IndexerOptions)
)

func buildOptions(opts ...IndexerOption) *IndexerOptions {
	options := &IndexerOptions{
		DbPath: "$HOME/.mm/chroma",
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithDbPath(dbPath string) func(*IndexerOptions) {
	return func(opts *IndexerOptions) {
		opts.DbPath = dbPath
	}
}

func RunIndexer(ctx context.Context, opts ...IndexerOption) error {
	logger := zerolog.Ctx(ctx)

	options := buildOptions(opts...)

	err := ensureDbPathExists(options.DbPath)
	if err != nil {
		logger.Error().Err(err).Msg("failed to ensure database path exists")
		return fmt.Errorf("failed to ensure database path exists: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "mm-embedding-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "indexer.py"), pythonScript, 0644); err != nil {
		return fmt.Errorf("failed to write Python script: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), pyprojectToml, 0644); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	log.Info().Str("indexer", filepath.Join(tmpDir, "indexer.py")).Msg("Running Python indexer with uv")
	cmdTokens := []string{
		"run",
		"python",
		"indexer.py",
	}
	args := buildIndexerCmdArgs(options)
	log.Info().Strs("args", cmdTokens).Msg("With tokens...")
	cmdTokens = append(cmdTokens, args...)

	cmd := exec.CommandContext(ctx, "uv", cmdTokens...)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("indexer failed: %w", err)
	}

	log.Info().Msg("Indexer completed successfully")
	return nil
}

func buildIndexerCmdArgs(options *IndexerOptions) []string {
	var args []string
	if options.DbPath != "" {
		args = append(args, "--db-path", options.DbPath)
	}

	return args
}

func ensureDbPathExists(path string) error {
	return os.MkdirAll(os.ExpandEnv(path), 0755)
}
