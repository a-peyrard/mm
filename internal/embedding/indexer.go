package embedding

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	libDirectoryName    = "lib"
	chromaDirectoryName = "chroma"
)

//go:embed python/indexer.py
var pythonScript []byte

//go:embed python/pyproject.toml
var pyprojectToml []byte

type (
	IndexerOptions struct {
		WorkingDirectory string
	}

	IndexerOption func(*IndexerOptions)
)

func buildOptions(opts ...IndexerOption) *IndexerOptions {
	options := &IndexerOptions{
		WorkingDirectory: "$HOME/.mm",
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func WithWorkingDirectory(wd string) func(*IndexerOptions) {
	return func(opts *IndexerOptions) {
		opts.WorkingDirectory = wd
	}
}

func RunIndexer(ctx context.Context, opts ...IndexerOption) error {
	logger := zerolog.Ctx(ctx)

	options := buildOptions(opts...)

	wd := os.ExpandEnv(options.WorkingDirectory)
	err := prepareWorkingDirectoryIfNeeded(ctx, os.ExpandEnv(wd))
	if err != nil {
		logger.Error().Err(err).Msg("failed to prepare working directory")
		return fmt.Errorf("failed to prepare working directory: %w", err)
	}

	cmdTokens := []string{
		"run",
		"python",
		"indexer.py",
	}
	cmdTokens = append(cmdTokens, buildIndexerCmdArgs(options)...)

	cmd := exec.CommandContext(ctx, "uv", cmdTokens...)
	cmd.Dir = filepath.Join(wd, libDirectoryName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Msg("running indexer")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("indexer failed: %w", err)
	}

	log.Info().Msg("Indexer completed successfully")
	return nil
}

func prepareWorkingDirectoryIfNeeded(ctx context.Context, wd string) error {
	logger := zerolog.Ctx(ctx)

	err := ensurePathExists(wd)
	if err != nil {
		logger.Error().Err(err).Msg("failed to ensure working directory exists")
		return fmt.Errorf("failed to ensure working directory exists: %w", err)
	}
	err = ensurePathExists(filepath.Join(wd, libDirectoryName))
	if err != nil {
		logger.Error().Err(err).Msg("failed to ensure lib directory exists")
		return fmt.Errorf("failed to ensure lib directory exists %w", err)
	}
	err = ensurePathExists(filepath.Join(wd, chromaDirectoryName))
	if err != nil {
		logger.Error().Err(err).Msg("failed to ensure database directory exists")
		return fmt.Errorf("failed to ensure database directory exists %w", err)
	}

	// Note: in the future we could generate checksums at compile time, and embed them in the binary,
	pythonScriptPath := filepath.Join(wd, libDirectoryName, "indexer.py")
	pyprojectTomlPath := filepath.Join(wd, libDirectoryName, "pyproject.toml")
	if requiresUpdate(pythonScriptPath, computeChecksum(pythonScript)) ||
		requiresUpdate(pyprojectTomlPath, computeChecksum(pyprojectToml)) {
		log.Debug().Msg("updating python script")

		_ = os.RemoveAll(filepath.Join(wd, libDirectoryName))
		err = ensurePathExists(filepath.Join(wd, libDirectoryName))
		if err != nil {
			logger.Error().Err(err).Msg("failed to ensure lib directory exists")
			return fmt.Errorf("failed to ensure lib directory exists %w", err)
		}

		err = os.WriteFile(pythonScriptPath, pythonScript, 0644)
		if err != nil {
			logger.Error().Err(err).Msg("failed to write python script")
			return fmt.Errorf("failed to write Python script: %w", err)
		}
		err = os.WriteFile(pythonScriptPath+".sha256", []byte(computeChecksum(pythonScript)), 0644)
		if err != nil {
			logger.Error().Err(err).Msg("failed to write python script checksum")
			return fmt.Errorf("failed to write Python script checksum: %w", err)
		}
		err = os.WriteFile(pyprojectTomlPath, pyprojectToml, 0644)
		if err != nil {
			logger.Error().Err(err).Msg("failed to write pyproject.toml")
			return fmt.Errorf("failed to write pyproject.toml: %w", err)
		}
		err = os.WriteFile(pyprojectTomlPath+".sha256", []byte(computeChecksum(pyprojectToml)), 0644)
		if err != nil {
			logger.Error().Err(err).Msg("failed to write pyproject.toml checksum")
			return fmt.Errorf("failed to write pyproject.toml checksum: %w", err)
		}
	}

	return nil
}

func buildIndexerCmdArgs(options *IndexerOptions) []string {
	var args []string
	if options.WorkingDirectory != "" {
		args = append(args, "--db-path", options.WorkingDirectory)
	}

	return args
}

func ensurePathExists(path string) error {
	return os.MkdirAll(path, 0755)
}

func computeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func requiresUpdate(path string, expectedSum string) bool {
	checksumFile := path + ".sha256"

	content, err := os.ReadFile(checksumFile)
	if err != nil {
		// the file does not exist, we need to update
		return true
	}

	return string(content) != expectedSum
}
