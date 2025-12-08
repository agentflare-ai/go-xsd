package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const version = "dev"

type globalOptions struct {
	logLevel            string
	strictContentModels bool
}

func newRootCommand() *cobra.Command {
	opts := &globalOptions{
		logLevel:            "info",
		strictContentModels: false,
	}

	cmd := &cobra.Command{
		Use:           "goxsd",
		Short:         "High-fidelity XSD tooling: validate XML, run W3C suites, generate docs",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return configureLogging(opts.logLevel)
		},
	}

	cmd.PersistentFlags().StringVar(&opts.logLevel, "log-level", opts.logLevel, "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().BoolVar(&opts.strictContentModels, "strict-content-model", opts.strictContentModels, "Enable strict content-model validation")

	cmd.AddCommand(
		newValidateCommand(opts),
		newTestCommand(opts),
		newDocCommand(opts),
	)

	return cmd
}

func configureLogging(level string) error {
	level = strings.ToLower(level)

	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn", "warning":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slogLevel}))
	slog.SetDefault(logger)
	return nil
}
