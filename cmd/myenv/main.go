package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/diff"
	"github.com/myenv-cli/myenv/internal/leaks"
	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
	"github.com/myenv-cli/myenv/internal/validate"
	"github.com/spf13/cobra"
)

const (
	red    = "\033[31m"
	yellow = "\033[33m"
	green  = "\033[32m"
	reset  = "\033[0m"
)

func main() {
	command := rootCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(command.ErrOrStderr(), "%serror:%s %v\nRun %q for commands and flags.\n", red, reset, err, "myenv help")
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:               "myenv",
		Short:             "Enforce environment configuration contracts",
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		Run: func(command *cobra.Command, arguments []string) {
			fmt.Fprintln(command.OutOrStdout(), "myenv checks environment configuration. Run 'myenv help' for commands and flags.")
		},
	}
	command.AddCommand(validateCommand(), scanCommand(), inferCommand())
	return command
}

func validateCommand() *cobra.Command {
	var schemaPath, envPath, format string
	command := &cobra.Command{Use: "validate", Short: "Validate a dotenv file against .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		rules, err := schema.Load(schemaPath)
		if err != nil {
			return err
		}
		values, err := validate.LoadDotenv(envPath)
		if err != nil {
			return err
		}
		return report(validate.Env(rules, values), format)
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path")
	command.Flags().StringVar(&envPath, "env", ".env", "dotenv path")
	command.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return command
}

func scanCommand() *cobra.Command {
	var schemaPath, root, envPath, format string
	command := &cobra.Command{Use: "scan", Short: "Cross-reference code, .env, and .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		rules, err := schema.Load(schemaPath)
		if err != nil {
			return err
		}
		if envPath == "" {
			envPath = filepath.Join(root, ".env")
		}
		values, err := validate.LoadDotenv(envPath)
		if err != nil {
			return err
		}
		accesses, diagnostics, err := scanner.Scan(root)
		if err != nil {
			return err
		}
		diagnostics = append(validate.Env(rules, values), diagnostics...)
		diagnostics = append(diagnostics, diff.Compare(rules, values, accesses)...)
		leakDiagnostics, err := leaks.ScanTracked(root)
		if err != nil {
			return err
		}
		diagnostics = append(diagnostics, leakDiagnostics...)
		return report(diagnostics, format)
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path")
	command.Flags().StringVar(&root, "root", ".", "repository root")
	command.Flags().StringVar(&envPath, "env", "", "dotenv path (defaults to <root>/.env)")
	command.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return command
}

func inferCommand() *cobra.Command {
	var envPath, output string
	command := &cobra.Command{Use: "infer", Short: "Generate a starter .myenv.yaml from a dotenv file", RunE: func(command *cobra.Command, arguments []string) error {
		values, err := validate.LoadDotenv(envPath)
		if err != nil {
			return err
		}
		rules := make(schema.Schema, len(values))
		for key, value := range values {
			rules[key] = schema.Rule{Key: key, Type: inferType(value), Secret: schema.LooksSecretName(key)}
		}
		contents, err := schema.Render(rules)
		if err != nil {
			return err
		}
		if err := os.WriteFile(output, contents, 0644); err != nil {
			return err
		}
		fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s created %s from %s\n", green, reset, output, envPath)
		return nil
	}}
	command.Flags().StringVar(&envPath, "env", ".env", "dotenv path")
	command.Flags().StringVar(&output, "output", ".myenv.yaml", "schema output path")
	return command
}

func inferType(value string) string {
	if _, err := strconv.ParseBool(value); err == nil {
		return "bool"
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil && strings.ContainsAny(value, ".eE") {
		return "float"
	}
	return "string"
}

func report(diagnostics []diagnostic.Diagnostic, format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported format %q", format)
	}
	if format == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(diagnostics); err != nil {
			return err
		}
	} else if len(diagnostics) == 0 {
		fmt.Printf("%s[PASS]%s no issues found\n", green, reset)
	} else {
		for _, item := range diagnostics {
			location := ""
			if item.Path != "" {
				location = filepath.Clean(item.Path)
				if item.Line != 0 {
					location += fmt.Sprintf(":%d", item.Line)
				}
				location += ": "
			}
			marker, color := "[WARN]", yellow
			if item.IsError() {
				marker, color = "[ERROR]", red
			}
			fmt.Printf("%s%s%s %s[%s] %s\n", color, marker, reset, location, item.Rule, item.Message)
		}
	}
	for _, item := range diagnostics {
		if item.IsError() {
			return fmt.Errorf("checks failed")
		}
	}
	return nil
}
