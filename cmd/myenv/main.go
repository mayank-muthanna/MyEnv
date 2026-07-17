package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/myenv-cli/myenv/internal/diagnostic"
	"github.com/myenv-cli/myenv/internal/diff"
	"github.com/myenv-cli/myenv/internal/envcrypt"
	"github.com/myenv-cli/myenv/internal/ignore"
	"github.com/myenv-cli/myenv/internal/infer"
	"github.com/myenv-cli/myenv/internal/leaks"
	"github.com/myenv-cli/myenv/internal/scanner"
	"github.com/myenv-cli/myenv/internal/schema"
	"github.com/myenv-cli/myenv/internal/validate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

const (
	red    = "\033[38;5;203m"
	yellow = "\033[38;5;220m"
	green  = "\033[38;5;114m"
	blue   = "\033[38;5;111m"
	gray   = "\033[38;5;245m"
	bold   = "\033[1m"
	reset  = "\033[0m"
)

var errChecksFailed = errors.New("checks failed")

func main() {
	command := rootCommand()
	if err := command.Execute(); err != nil {
		if !errors.Is(err, errChecksFailed) {
			fmt.Fprintf(command.ErrOrStderr(), "%s%s[ERROR]%s %v\n%s[HINT]%s Run %q for commands and flags.\n", bold, red, reset, err, blue, reset, "myenv help")
		}
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
	command.AddCommand(validateCommand(), scanCommand(), inferCommand(), encryptCommand(), decryptCommand())
	command.SetHelpFunc(renderHelp)
	return command
}

func renderHelp(command *cobra.Command, _ []string) {
	output := command.OutOrStdout()
	if command == command.Root() {
		renderRootHelp(output)
		return
	}
	renderCommandHelp(output, command)
}

func renderRootHelp(output io.Writer) {
	section(output, "MYENV", "Environment configuration guardrails")
	line(output, gray, "Keep code, dotenv files, and .myenv.yaml in sync.")
	fmt.Fprintln(output)
	section(output, "START HERE", "Three commands. One normal flow.")
	step(output, "1", "Create schema", "myenv infer")
	step(output, "2", "Validate values", "myenv validate")
	step(output, "3", "Scan project drift", "myenv scan")
	fmt.Fprintln(output)
	section(output, "COMMANDS", "What each command checks")
	helpCommand(output, "infer", "Generate or sync .myenv.yaml from a dotenv file", "--env, --output")
	helpCommand(output, "validate", "Check dotenv values against schema rules", "--env, --schema, --format")
	helpCommand(output, "scan", "Cross-reference code, dotenv, and schema", "--root, --env, --schema, --format")
	helpCommand(output, "encrypt", "Compress and encrypt a dotenv file into schema", "--env, --schema, --key")
	helpCommand(output, "decrypt", "Restore encrypted dotenv with its key", "--schema, --key, --output")
	fmt.Fprintln(output)
	section(output, "NEED DETAILS?", "Every command has its own flags and examples")
	commandExample(output, "myenv help encrypt")
	commandExample(output, "myenv help scan")
	commandExample(output, "myenv validate --help")
}

func renderCommandHelp(output io.Writer, command *cobra.Command) {
	title := "MYENV " + strings.ToUpper(command.Name())
	section(output, title, command.Short)
	line(output, gray, commandGuidance(command.Name()))
	fmt.Fprintln(output)
	section(output, "USAGE", "")
	commandExample(output, command.UseLine())
	fmt.Fprintln(output)
	section(output, "EXAMPLES", "")
	for _, example := range commandExamples(command.Name()) {
		commandExample(output, example)
	}
	fmt.Fprintln(output)
	section(output, "FLAGS", "")
	flags := command.NonInheritedFlags()
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "help" {
			return
		}
		name := "--" + flag.Name
		if flag.Shorthand != "" {
			name = "-" + flag.Shorthand + ", " + name
		}
		defaultValue := ""
		if flag.DefValue != "" && flag.DefValue != "false" {
			defaultValue = " (default: " + flag.DefValue + ")"
		}
		fmt.Fprintf(output, "  %s%-22s%s %s%s%s%s\n", blue, name, reset, flag.Usage, gray, defaultValue, reset)
	})
	fmt.Fprintf(output, "  %s%-22s%s %s\n", blue, "-h, --help", reset, "Show this command guide.")
	fmt.Fprintln(output)
	line(output, gray, "Tip: use --format json in CI or scripts.")
}

func commandGuidance(name string) string {
	switch name {
	case "infer":
		return "Use after creating or changing a dotenv file. Existing schemas offer Override, Sync, or Skip."
	case "validate":
		return "Use before running your app to catch missing, malformed, or unsafe configuration."
	case "scan":
		return "Use in development or CI to find config drift, unused values, and likely committed secrets."
	case "encrypt":
		return "Use to store a compressed, encrypted dotenv payload inside .myenv.yaml. Save printed key outside repository."
	case "decrypt":
		return "Use saved encryption key to restore encryptedEnv. Existing output files require --force."
	default:
		return "Run this command with the flags below."
	}
}

func commandExamples(name string) []string {
	switch name {
	case "infer":
		return []string{"myenv infer --env .env.local", "myenv infer --env .env.local --output config/.myenv.yaml"}
	case "validate":
		return []string{"myenv validate --env .env.local", "myenv validate --schema config/.myenv.yaml --env .env.production", "myenv validate --env .env.local --format json"}
	case "scan":
		return []string{"myenv scan --root . --env .env.local", "myenv scan --root apps/web --schema config/.myenv.yaml --env apps/web/.env", "myenv scan --env .env.local --format json"}
	case "encrypt":
		return []string{"myenv encrypt", "myenv encrypt --env .env.local", "myenv encrypt --key <base64url-32-byte-key>"}
	case "decrypt":
		return []string{"myenv decrypt --key <saved-key>", "myenv decrypt --key <saved-key> --output .env --force"}
	default:
		return nil
	}
}
func section(output io.Writer, title, subtitle string) {
	fmt.Fprintf(output, "%s%s%s%s", bold, blue, title, reset)
	if subtitle != "" {
		fmt.Fprintf(output, "  %s%s%s", gray, subtitle, reset)
	}
	fmt.Fprintln(output)
	fmt.Fprintf(output, "%s------------------------------------------------------------%s\n", gray, reset)
}

func line(output io.Writer, color, message string) {
	fmt.Fprintf(output, "%s%s%s\n", color, message, reset)
}

func step(output io.Writer, number, label, example string) {
	fmt.Fprintf(output, "  %s%s.%s %-18s %s%s%s\n", bold, number, reset, label, green, example, reset)
}

func helpCommand(output io.Writer, name, description, flags string) {
	fmt.Fprintf(output, "  %s%-10s%s %-49s %s%s%s\n", green, name, reset, description, gray, flags, reset)
}

func commandExample(output io.Writer, example string) {
	fmt.Fprintf(output, "  %s$%s %s%s%s\n", gray, reset, green, example, reset)
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
		return report("validate", validate.Env(rules, values), format)
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path")
	command.Flags().StringVar(&envPath, "env", ".env", "dotenv path")
	command.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return command
}

func scanCommand() *cobra.Command {
	var schemaPath, root, envPath, format string
	command := &cobra.Command{Use: "scan", Short: "Cross-reference code, .env, and .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		document, err := schema.LoadDocument(schemaPath)
		if err != nil {
			return err
		}
		rules := document.Schema
		if envPath == "" {
			envPath = filepath.Join(root, ".env")
		}
		policy := ignore.Config{Code: document.IgnoreCode, Unused: document.IgnoreUnused, Paths: document.IgnorePaths, Rules: document.IgnoreRules}
		values, err := validate.LoadDotenv(envPath)
		if err != nil {
			return err
		}
		accesses, diagnostics, err := scanner.Scan(root, policy)
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
		diagnostics = policy.Filter(root, diagnostics)
		return report("scan", diagnostics, format)
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path")
	command.Flags().StringVar(&root, "root", ".", "repository root")
	command.Flags().StringVar(&envPath, "env", "", "dotenv path (defaults to <root>/.env)")
	command.Flags().StringVar(&format, "format", "text", "output format: text or json")
	return command
}

func encryptCommand() *cobra.Command {
	var schemaPath, envPath, encodedKey string
	command := &cobra.Command{Use: "encrypt", Short: "Compress and encrypt a dotenv file into .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		plaintext, err := os.ReadFile(envPath)
		if err != nil {
			return err
		}
		document, err := schema.LoadDocument(schemaPath)
		if err != nil {
			return err
		}
		key, generated, err := encryptionKey(encodedKey)
		if err != nil {
			return err
		}
		payload, err := envcrypt.Encrypt(plaintext, key)
		if err != nil {
			return err
		}
		document.EncryptedEnv = payload
		contents, err := schema.RenderDocument(document)
		if err != nil {
			return err
		}
		if err := os.WriteFile(schemaPath, contents, 0600); err != nil {
			return err
		}
		fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s encrypted %s into %s %s(gzip + AES-256-GCM)%s\n", green, reset, envPath, schemaPath, gray, reset)
		if generated {
			encoded, err := envcrypt.EncodeKey(key)
			if err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "%s[KEY]%s Save this key outside repository: %s%s%s\n", bold, yellow, green, encoded, reset)
		}
		fmt.Fprintf(command.OutOrStdout(), "%s[HINT]%s Key never saved in %s. Use %q to restore exact dotenv file.\n", blue, reset, schemaPath, "myenv decrypt --key <saved-key>")
		return nil
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path where encryptedEnv is stored")
	command.Flags().StringVar(&envPath, "env", ".env", "dotenv file to encrypt")
	command.Flags().StringVar(&encodedKey, "key", "", "optional base64url-encoded 32-byte key; omit to generate one")
	return command
}

func decryptCommand() *cobra.Command {
	var schemaPath, output, encodedKey string
	var force bool
	command := &cobra.Command{Use: "decrypt", Short: "Restore encrypted dotenv values from .myenv.yaml", RunE: func(command *cobra.Command, arguments []string) error {
		if encodedKey == "" {
			return fmt.Errorf("--key is required; myenv never stores encryption keys")
		}
		key, err := envcrypt.ParseKey(encodedKey)
		if err != nil {
			return err
		}
		document, err := schema.LoadDocument(schemaPath)
		if err != nil {
			return err
		}
		plaintext, err := envcrypt.Decrypt(document.EncryptedEnv, key)
		if err != nil {
			return err
		}
		if _, err := os.Stat(output); err == nil && !force {
			return fmt.Errorf("%s already exists; choose another --output or add --force", output)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := os.WriteFile(output, plaintext, 0600); err != nil {
			return err
		}
		fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s restored encrypted dotenv to %s\n", green, reset, output)
		return nil
	}}
	command.Flags().StringVar(&schemaPath, "schema", ".myenv.yaml", "schema path containing encryptedEnv")
	command.Flags().StringVar(&encodedKey, "key", "", "base64url-encoded 32-byte key printed or supplied during encrypt")
	command.Flags().StringVar(&output, "output", ".env.decrypted", "dotenv output path")
	command.Flags().BoolVar(&force, "force", false, "replace an existing output file")
	return command
}

func encryptionKey(encoded string) ([]byte, bool, error) {
	if encoded != "" {
		key, err := envcrypt.ParseKey(encoded)
		return key, false, err
	}
	key, err := envcrypt.GenerateKey()
	return key, true, err
}
func inferCommand() *cobra.Command {
	var envPath, output string
	command := &cobra.Command{Use: "infer", Short: "Generate or sync .myenv.yaml from a dotenv file", RunE: func(command *cobra.Command, arguments []string) error {
		values, err := validate.LoadDotenv(envPath)
		if err != nil {
			return err
		}
		rules := make(schema.Schema, len(values))
		for key, value := range values {
			rules[key] = schema.Rule{Key: key, Type: inferType(value), Secret: schema.LooksSecretName(key)}
		}

		document := schema.Document{Schema: rules}
		mode := "created"
		change := infer.Change{Added: len(rules)}
		if _, err := os.Stat(output); err == nil {
			choice, err := chooseInferAction(command.InOrStdin(), command.OutOrStdout(), output)
			if err != nil {
				return err
			}
			if choice == inferSkip {
				fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s kept %s unchanged\n", green, reset, output)
				return nil
			}
			if choice == inferSync {
				existing, err := schema.LoadDocument(output)
				if err != nil {
					return err
				}
				document = existing
				document.Schema, change = infer.Merge(existing.Schema, rules)
				mode = "synced"
			} else {
				mode = "overrode"
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		contents, err := schema.RenderDocument(document)
		if err != nil {
			return err
		}
		if err := os.WriteFile(output, contents, 0644); err != nil {
			return err
		}
		if mode == "synced" {
			fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s synced %s from %s %s(%d added, %d removed, %d preserved)%s\n", green, reset, output, envPath, gray, change.Added, change.Removed, change.Preserved, reset)
		} else {
			fmt.Fprintf(command.OutOrStdout(), "%s[PASS]%s %s %s from %s\n", green, reset, mode, output, envPath)
		}
		return nil
	}}
	command.Flags().StringVar(&envPath, "env", ".env", "dotenv path")
	command.Flags().StringVar(&output, "output", ".myenv.yaml", "schema output path")
	return command
}

type inferAction int

const (
	inferOverride inferAction = iota
	inferSync
	inferSkip
)

var inferActions = []string{
	"Override current schema completely",
	"Sync additions and removals; preserve existing rule settings",
	"Skip; keep current schema unchanged",
}

func chooseInferAction(input io.Reader, output io.Writer, path string) (inferAction, error) {
	file, isFile := input.(*os.File)
	if !isFile || !term.IsTerminal(int(file.Fd())) {
		return inferSkip, fmt.Errorf("%s already exists; run infer in an interactive terminal to choose override, sync, or skip", path)
	}
	oldState, err := term.MakeRaw(int(file.Fd()))
	if err != nil {
		return inferSkip, fmt.Errorf("enable interactive input: %w", err)
	}
	defer term.Restore(int(file.Fd()), oldState)

	selected := 1
	renderInferMenu(output, path, selected)
	buffer := make([]byte, 1)
	for {
		if _, err := file.Read(buffer); err != nil {
			return inferSkip, err
		}
		switch buffer[0] {
		case 3:
			return inferSkip, fmt.Errorf("selection cancelled")
		case '\r', '\n':
			return inferAction(selected), nil
		case ' ':
			selected = (selected + 1) % len(inferActions)
			renderInferMenu(output, path, selected)
		case 27:
			sequence := make([]byte, 2)
			if _, err := io.ReadFull(file, sequence); err != nil || sequence[0] != '[' {
				continue
			}
			if sequence[1] == 'A' {
				selected = (selected + len(inferActions) - 1) % len(inferActions)
				renderInferMenu(output, path, selected)
			}
			if sequence[1] == 'B' {
				selected = (selected + 1) % len(inferActions)
				renderInferMenu(output, path, selected)
			}
		}
	}
}

func renderInferMenu(output io.Writer, path string, selected int) {
	fmt.Fprint(output, "\033[2J\033[H")
	fmt.Fprintf(output, "%s%sMYENV INFER%s\n%s%s already exists. Choose action:%s\n\n", bold, blue, reset, gray, path, reset)
	for index, action := range inferActions {
		marker, color := "[ ]", gray
		if index == selected {
			marker, color = "[x]", blue
		}
		fmt.Fprintf(output, "%s%s %s%s\n", color, marker, action, reset)
	}
	fmt.Fprintf(output, "\n%sUp/down arrows or Space: move. Enter: select. Ctrl+C: cancel.%s", gray, reset)
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

func report(commandName string, diagnostics []diagnostic.Diagnostic, format string) error {
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
		fmt.Printf("%s%s[PASS]%s %s%s completed with no issues.%s\n", bold, green, reset, gray, strings.ToUpper(commandName), reset)
	} else {
		errorCount, warningCount := 0, 0
		for _, item := range diagnostics {
			if item.IsError() {
				errorCount++
			} else {
				warningCount++
			}
		}

		fmt.Printf("%s%sMYENV %s%s  %s%d diagnostics%s\n", bold, blue, strings.ToUpper(commandName), reset, gray, len(diagnostics), reset)
		fmt.Printf("%s------------------------------------------------------------%s\n", gray, reset)
		for _, item := range diagnostics {
			label, color := "[WARN]", yellow
			if item.IsError() {
				label, color = "[ERROR]", red
			}
			location := ""
			if item.Path != "" {
				location = filepath.Clean(item.Path)
				if item.Line != 0 {
					location += fmt.Sprintf(":%d", item.Line)
				}
			}
			fmt.Printf("%s%s%s %s%s%s  %s%s", bold, color, label, reset, blue, item.Rule, reset, item.Message)
			if location != "" {
				fmt.Printf("  %s@ %s%s", gray, location, reset)
			}
			fmt.Println()
			if item.Hint != "" {
				fmt.Printf("  %shint: %s%s\n", gray, item.Hint, reset)
			}
		}
		fmt.Printf("%s------------------------------------------------------------%s\n", gray, reset)
		if errorCount > 0 {
			fmt.Printf("%s%s[FAIL]%s %d errors, %d warnings. %s[HINT]%s Run %q for commands and flags.\n", bold, red, reset, errorCount, warningCount, blue, reset, "myenv help")
			return errChecksFailed
		}
		fmt.Printf("%s%s[PASS]%s no errors, %d warnings.\n", bold, green, reset, warningCount)
	}
	return nil
}
