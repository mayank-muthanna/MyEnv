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

func main() {
	if err := rootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	command := &cobra.Command{Use: "myenv", Short: "Enforce environment configuration contracts"}
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
	var schemaPath, root, format string
	command := &cobra.Command{Use: "scan", Short: "Cross-reference source environment usage with .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		rules, err := schema.Load(schemaPath)
		if err != nil {
			return err
		}
		accesses, diagnostics, err := scanner.Scan(root)
		if err != nil {
			return err
		}
		diagnostics = append(diagnostics, diff.Compare(rules, accesses)...)
		leakDiagnostics, err := leaks.ScanTracked(root)
		if err != nil {
			return err
		}
		diagnostics = append(diagnostics, leakDiagnostics...)
		return report(diagnostics, format)
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path")
	command.Flags().StringVar(&root, "root", ".", "repository root")
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
		fmt.Fprintf(command.OutOrStdout(), "created %s from %s\n", output, envPath)
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
		fmt.Println("✓ no issues found")
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
			marker := "⚠"
			if item.IsError() {
				marker = "✗"
			}
			fmt.Printf("%s %s[%s] %s\n", marker, location, item.Rule, item.Message)
		}
	}
	for _, item := range diagnostics {
		if item.IsError() {
			return fmt.Errorf("checks failed")
		}
	}
	return nil
}
