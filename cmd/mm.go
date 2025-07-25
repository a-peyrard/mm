package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/a-peyrard/mm/internal/code"
	"github.com/a-peyrard/mm/internal/embedding"
	"github.com/a-peyrard/mm/internal/set"
	"github.com/a-peyrard/mm/internal/worker"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	index bool
)

var mmCmd = &cobra.Command{
	Use:   "mm --index [file ...]",
	Short: "My Memory CLI tool",
	Long:  `My Memory CLI tool`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var writer io.Writer = zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
		logger := zerolog.New(writer).
			With().
			Timestamp().
			Caller().
			Logger()
		ctx := logger.WithContext(cmd.Context())

		if index {
			workerGroup, err := worker.NewGroup(ctx, 12, NewIndexerWorker)
			if err != nil {
				return fmt.Errorf("failed to create worker group: %w", err)
			}

			// look for Python files in the provided directory
			path := args[0]
			err = code.FindInDirectory(
				path,
				set.Of(".py"),
				workerGroup.Submit,
			)
			if err != nil {
				return fmt.Errorf("failed to find files in directory %s: %w", path, err)
			}

			_ = workerGroup.WaitAndClose()
		}

		return nil
	},
}

type indexerWorker struct {
	indexer *embedding.RunningIndexer
}

func NewIndexerWorker(ctx context.Context, workerIdx int) (worker.Worker[string], error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("process", "python indexer").
		Int("workerIdx", workerIdx).
		Logger()

	// create the embedding indexer
	indexer, err := embedding.RunIndexer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run indexer: %w", err)
	}
	go func() {
		for out := range indexer.Output() {
			logger.Debug().Msg(out)
		}
	}()

	return &indexerWorker{indexer}, nil
}

func (i *indexerWorker) WaitReady(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (i *indexerWorker) Handle(_ context.Context, filePath string) error {
	log.Debug().Str("path", filePath).Msg("Processing file")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	chunks, err := code.NewGenericParser().ParseFile(filePath, content)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}
	if len(chunks) > 0 {
		err = i.indexer.ProcessChunk(chunks)
		if err != nil {
			return fmt.Errorf("failed to process chunk: %w", err)
		}
	}

	return nil
}

func (i *indexerWorker) WaitAndClose() error {
	return i.indexer.WaitAndClose()
}

func init() {
	mmCmd.Flags().BoolVar(
		&index,
		"index",
		false,
		"If we should run in index mode (otherwise will run in consume mode)",
	)
}

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		Level(zerolog.TraceLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	if err := mmCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
