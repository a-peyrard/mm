package embedding

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/a-peyrard/mm/internal/code"
	"github.com/rs/zerolog"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

	RunningIndexer struct {
		ctx    context.Context
		logger *zerolog.Logger

		command *exec.Cmd

		stdin io.WriteCloser

		stdout io.ReadCloser
		stderr io.ReadCloser

		out          chan string
		completionCh chan struct{}

		pendingChunks *atomic.Int32

		ready *sync.WaitGroup
	}
)

func WithWorkingDirectory(wd string) func(*IndexerOptions) {
	return func(opts *IndexerOptions) {
		opts.WorkingDirectory = wd
	}
}

func RunIndexer(ctx context.Context, opts ...IndexerOption) (*RunningIndexer, error) {
	logger := zerolog.Ctx(ctx)

	options := buildOptions(opts...)

	wd := os.ExpandEnv(options.WorkingDirectory)
	err := prepareWorkingDirectoryIfNeeded(ctx, os.ExpandEnv(wd))
	if err != nil {
		logger.Error().Err(err).Msg("failed to prepare working directory")
		return nil, fmt.Errorf("failed to prepare working directory: %w", err)
	}

	cmdTokens := []string{
		"run",
		"python",
		"indexer.py",
	}
	// fixme: we will need to pass the db path to the chroma server, and run it somewhere else
	// cmdTokens = append(cmdTokens, buildIndexerCmdArgs(wd)...)

	cmd := exec.CommandContext(ctx, "uv", cmdTokens...)
	cmd.Dir = filepath.Join(wd, libDirectoryName)

	// Set up pipes for communication
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	runningIndexer := initRunningIndexer(ctx, cmd, stdin, stdout, stderr)

	logger.Trace().Msg("running indexer sub-process")
	if err := cmd.Start(); err != nil {
		_ = runningIndexer.Close()
		return nil, fmt.Errorf("indexer failed: %w", err)
	}

	return runningIndexer, nil
}

func initRunningIndexer(ctx context.Context, cmd *exec.Cmd, stdin io.WriteCloser, stdout io.ReadCloser, stderr io.ReadCloser) *RunningIndexer {
	logger := zerolog.Ctx(ctx)

	out := captureOutput(ctx, stdout, stderr, logger)

	completionCh := make(chan struct{})

	ready := sync.WaitGroup{}
	ready.Add(1)
	pendingChunks := atomic.Int32{}
	outWrapped := make(chan string)
	go func() {
		defer close(outWrapped)
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-out:
				if !ok {
					return
				}

				select {
				case outWrapped <- line:
				case <-ctx.Done():
					return
					// fixme: restore this or another mechanism to not hang if no-one is listening.
					//   but if we put this, the listener is missing some of the logs
					//default:
					//	// maybe no one is reading the output, so we just drop it
				}

				if !strings.Contains(line, "status") {
					continue
				}

				if strings.Contains(line, "READY") {
					ready.Done()
				}

				val := pendingChunks.Add(-1)
				if val < 0 {
					// don't want negative values, this counter is not precise science, we would need to
					// identify the chunks sent with some unique ids, here we just assume that the indexer
					// is always returning a single line per chunk processed
					pendingChunks.CompareAndSwap(val, 0)
				}

				if val <= 0 {
					select {
					case completionCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}()

	return &RunningIndexer{
		ctx:    ctx,
		logger: logger,

		command: cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,

		out:          outWrapped,
		completionCh: completionCh,

		pendingChunks: &pendingChunks,

		ready: &ready,
	}
}

func captureOutput(ctx context.Context, stdout io.ReadCloser, stderr io.ReadCloser, logger *zerolog.Logger) chan string {
	out := make(chan string)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				select {
				case <-ctx.Done():
					return
				case out <- line:
				}
			}
		}
		if err := scanner.Err(); err != nil && !strings.Contains(err.Error(), "closed") {
			logger.Error().Err(err).Msg("error reading stdout")
		}
	}()
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				select {
				case <-ctx.Done():
					return
				case out <- line:
				}
			}
		}
		if err := scanner.Err(); err != nil && !strings.Contains(err.Error(), "closed") {
			logger.Error().Err(err).Msg("error reading stderr")
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func (i *RunningIndexer) WaitReady() error {
	i.ready.Wait()

	return nil
}

func (i *RunningIndexer) Output() <-chan string {
	return i.out
}

func (i *RunningIndexer) ProcessChunk(chunks []code.Chunk) error {
	toProcess := map[string]any{
		"chunks": chunks,
	}
	bytes, err := json.Marshal(toProcess)
	if err != nil {
		i.logger.Error().Err(err).Msg("failed to marshal chunks")
		return fmt.Errorf("failed to marshal chunks: %w", err)
	}

	i.pendingChunks.Add(1)
	_, err = fmt.Fprintln(i.stdin, string(bytes))
	if err != nil {
		i.pendingChunks.Add(-1)
		i.logger.Error().Err(err).Msg("failed to write chunks to stdin")
		return fmt.Errorf("failed to write chunks to stdin: %w", err)
	}

	return nil
}

func (i *RunningIndexer) WaitForCompletion() {
	i.logger.Trace().Msg("wait for completion of indexer")
	if i.pendingChunks.Load() == 0 {
		return
	}

	select {
	case <-i.ctx.Done():
	case <-i.completionCh:
	}

	return
}

func (i *RunningIndexer) Close() error {
	i.logger.Trace().Msg("close indexer")
	var errs []error

	if err := i.stdin.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
	}
	if err := i.command.Process.Kill(); err != nil {
		errs = append(errs, fmt.Errorf("failed to kill process: %w", err))
	}
	if err := i.stdout.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
	}
	if err := i.stderr.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close stdin: %w", err))
	}

	return errors.Join(errs...)
}

func (i *RunningIndexer) WaitAndClose() error {
	i.WaitForCompletion()
	return i.Close()
}

func buildOptions(opts ...IndexerOption) *IndexerOptions {
	options := &IndexerOptions{
		WorkingDirectory: "$HOME/.mm",
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
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
		logger.Debug().Msg("updating python script")

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

func buildIndexerCmdArgs(wd string) []string {
	return []string{
		"--db-path",
		filepath.Join(wd, chromaDirectoryName),
	}
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
