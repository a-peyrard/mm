package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/a-peyrard/mm/internal/code"
	"github.com/a-peyrard/mm/internal/embedding"
	"github.com/a-peyrard/mm/internal/set"
	"github.com/a-peyrard/mm/internal/worker"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	index           bool
	numberOfWorkers int
)

const defaultNumberOfWorkers = 2
const defaultLogLevel = zerolog.DebugLevel

var mmCmd = &cobra.Command{
	Use:   "mm --index [file ...]",
	Short: "My Memory CLI tool",
	Long:  `My Memory CLI tool`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "completion" {
			shell := "zsh"
			if len(args) > 1 {
				shell = args[1]
			}
			return handleCompletion(cmd, shell)
		}

		logger := log.Logger.
			With().
			Timestamp().
			Caller().
			Logger()
		ctx := logger.WithContext(cmd.Context())

		if index {
			logger.Info().Int("numberOfWorkers", numberOfWorkers).Msg("Initializing indexer daemons...")
			start := time.Now()
			workerGroup, err := worker.NewGroup(ctx, numberOfWorkers, NewIndexerWorker)
			if err != nil {
				return fmt.Errorf("failed to create worker group: %w", err)
			}
			_ = workerGroup.WaitAllWorkersToBeReady(ctx)
			end := time.Now()
			logger.Info().
				Str("elapsed", fmt.Sprintf("%dms", end.Sub(start).Milliseconds())).
				Int("numberOfWorkers", numberOfWorkers).
				Msg("daemons ready")

			// look for Python files in the provided directory
			start = time.Now()
			counter := 0
			path := args[0]
			err = code.FindInDirectory(
				path,
				set.Of(".py"),
				func(path string) error {
					counter++
					return workerGroup.Submit(path)
				},
			)
			if err != nil {
				return fmt.Errorf("failed to find files in directory %s: %w", path, err)
			}

			_ = workerGroup.WaitAndClose()
			end = time.Now()

			logger.Info().
				Str("elapsed", fmt.Sprintf("%dms", end.Sub(start).Milliseconds())).
				Int("filesProcessed", counter).
				Msg("Indexing completed")
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
			logger.Trace().Msg(out)
		}
	}()

	return &indexerWorker{indexer}, nil
}

func (w *indexerWorker) WaitReady(ctx context.Context) error {
	return w.indexer.WaitReady()
}

func (w *indexerWorker) Handle(_ context.Context, filePath string) error {
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
		err = w.indexer.ProcessChunk(chunks)
		if err != nil {
			return fmt.Errorf("failed to process chunk: %w", err)
		}
		w.indexer.WaitForCompletion()
	}

	return nil
}

func (w *indexerWorker) WaitAndClose() error {
	return w.indexer.Close()
}

func init() {
	mmCmd.Flags().BoolVar(
		&index,
		"index",
		false,
		"If we should run in index mode (otherwise will run in consume mode)",
	)

	mmCmd.Flags().IntVarP(
		&numberOfWorkers,
		"number-of-workers",
		"n",
		defaultNumberOfWorkers,
		fmt.Sprintf("Number of workers to use for indexing (default is %d)", defaultNumberOfWorkers),
	)

	mmCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("number-of-workers") && !index {
			return fmt.Errorf("--number-of-workers can only be used with --index")
		}
		return nil
	}
}

func handleCompletion(cmd *cobra.Command, shell string) error {
	switch shell {
	case "bash":
		return cmd.GenBashCompletion(os.Stdout)
	case "zsh":
		return cmd.GenZshCompletion(os.Stdout)
	case "fish":
		return cmd.GenFishCompletion(os.Stdout, true)
	default:
		return cmd.Help()
	}
}

func getLogLevel() zerolog.Level {
	return getLogLevelFromEnv("LOG_LEVEL", defaultLogLevel)
}

func getLogLevelFromEnv(envName string, fallbackLevel zerolog.Level) zerolog.Level {
	env := os.Getenv(envName)
	if env == "" {
		return fallbackLevel
	}
	level, err := zerolog.ParseLevel(env)
	if err != nil {
		fmt.Printf("Unable to parse log level from environment variable %s: '%s': %v\n", envName, env, err)
		level = fallbackLevel
	}

	return level
}

func main() {
	zerolog.SetGlobalLevel(getLogLevel())
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
