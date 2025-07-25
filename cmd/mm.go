package main

import (
	_ "embed"
	"fmt"
	"github.com/a-peyrard/mm/internal/code"
	"github.com/a-peyrard/mm/internal/embedding"
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
			filePath := args[0]
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			chunks, err := code.NewGenericParser().ParseFile(filePath, content)
			if err != nil {
				return fmt.Errorf("failed to parse file %s: %w", filePath, err)
			}

			indexer, err := embedding.RunIndexer(ctx)
			if err != nil {
				return fmt.Errorf("failed to run indexer: %w", err)
			}
			go func() {
				logger := logger.With().Str("process", "python indexer").Logger()
				for out := range indexer.Output() {
					logger.Debug().Msg(out)
				}
			}()

			err = indexer.ProcessChunk(chunks)
			if err != nil {
				return fmt.Errorf("failed to process chunk: %w", err)
			}

			err = indexer.WaitAndClose()
			if err != nil {
				return fmt.Errorf("failed to wait for indexer completion: %w", err)
			}
		}

		return nil
	},
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
