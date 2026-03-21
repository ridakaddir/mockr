package cmd

import (
	"fmt"

	"github.com/ridakaddir/mockr/internal/generate"
	"github.com/spf13/cobra"
)

var (
	genSpec   string
	genOut    string
	genFormat string
	genSplit  bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a mockr config directory from an OpenAPI spec",
	Long: `Generate reads an OpenAPI 3 spec (file or URL) and produces:
  - A mockr config file (TOML/YAML/JSON) with one route per operation
  - Stub JSON files in stubs/ populated from spec examples or synthesised from schemas

Examples:
  mockr generate --spec openapi.yaml --out ./mocks
  mockr generate --spec https://petstore3.swagger.io/api/v3/openapi.json --out ./mocks
  mockr generate --spec openapi.yaml --format yaml --split=false`,
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&genSpec, "spec", "s", "", "OpenAPI spec file path or URL (required)")
	generateCmd.Flags().StringVarP(&genOut, "out", "o", "mocks", "Output directory for config files and stubs")
	generateCmd.Flags().StringVarP(&genFormat, "format", "f", "toml", "Config format: toml, yaml, json")
	generateCmd.Flags().BoolVar(&genSplit, "split", true, "Split routes into one file per tag (use --split=false for a single file)")

	_ = generateCmd.MarkFlagRequired("spec")

	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	opts := generate.Options{
		Spec:   genSpec,
		OutDir: genOut,
		Format: genFormat,
		Split:  genSplit,
	}

	fmt.Printf("Loading spec: %s\n", genSpec)

	result, err := generate.Run(opts)
	if err != nil {
		return fmt.Errorf("generate failed: %w", err)
	}

	fmt.Printf("\nSpec: %s %s\n", result.SpecTitle, result.SpecVersion)
	fmt.Printf("Routes generated: %d\n", result.Routes)
	fmt.Printf("Stub files: %d\n", len(result.StubFiles))
	fmt.Printf("\nConfig files written:\n")
	for _, f := range result.ConfigFiles {
		fmt.Printf("  %s\n", f)
	}
	fmt.Printf("\nStub files written to: %s/stubs/\n", genOut)
	fmt.Printf("\nRun: mockr --config %s\n", genOut)

	return nil
}
