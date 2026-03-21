package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ridakaddir/mockr/internal/logger"
	"github.com/ridakaddir/mockr/internal/proxy"
	"github.com/spf13/cobra"
)

// defaultConfig resolves the config path to use when --config is not set:
//   - "./mockr.toml" if it exists (single-file default)
//   - "."            otherwise (load all config files in current directory)
func defaultConfig() string {
	if _, err := os.Stat("mockr.toml"); err == nil {
		return "mockr.toml"
	}
	return "."
}

const version = "0.1.0"

var (
	target     string
	port       int
	configFile string
	apiPrefix  string
	recordMode bool
	initMode   bool
)

var rootCmd = &cobra.Command{
	Use:     "mockr",
	Short:   "mockr — mock, stub and proxy APIs for frontend development",
	Version: version,
	RunE:    run,
}

// Execute is the entrypoint called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&target, "target", "t", "", "Upstream API URL to proxy unmatched requests to")
	rootCmd.Flags().IntVarP(&port, "port", "p", 4000, "Port to listen on")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file or directory (default: mockr.toml if present, else current directory)")
	rootCmd.Flags().StringVarP(&apiPrefix, "api-prefix", "a", "", "Strip this prefix from request paths before matching routes and forwarding to upstream (e.g. /api)")
	rootCmd.Flags().BoolVar(&recordMode, "record", false, "Record mode: proxy all requests and save responses as stubs")
	rootCmd.Flags().BoolVar(&initMode, "init", false, "Scaffold a mockr.toml template in the current directory")
}

func run(cmd *cobra.Command, args []string) error {
	// --init: scaffold config files and exit.
	if initMode {
		if err := proxy.Init("."); err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		logger.Info("scaffolded mockr.toml and stubs/users.json")
		logger.Info("run: mockr --target https://api.example.com")
		return nil
	}

	// Resolve config path: explicit flag > mockr.toml > current directory.
	cfg := configFile
	if cfg == "" {
		cfg = defaultConfig()
	}

	// Require --target if record mode is on (nothing to proxy otherwise).
	if recordMode && target == "" {
		return fmt.Errorf("--record requires --target")
	}

	opts := proxy.ServerOptions{
		Target:     target,
		Port:       port,
		ConfigPath: cfg,
		ApiPrefix:  apiPrefix,
		RecordMode: recordMode,
	}

	srv, err := proxy.NewServer(opts)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return srv.Start(ctx)
}
