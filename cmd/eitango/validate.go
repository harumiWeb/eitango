package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/dict"
)

const (
	validateKindCore   = "core"
	validateKindImport = "import"
	formatAuto         = "auto"
	formatCSV          = "csv"
	formatJSONL        = "jsonl"
)

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate dictionary files or embedded core words",
		Args:  cobra.NoArgs,
		RunE:  runValidate,
	}
	cmd.Flags().String("file", "", "Dictionary file to validate (.csv or .jsonl)")
	cmd.Flags().String("format", formatAuto, "Input format (auto, csv, jsonl)")
	cmd.Flags().String("kind", "", "Validation kind (core or import)")
	cmd.Flags().Bool("embedded-core", false, "Validate the embedded core dictionary")
	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	input, err := resolveValidationInput(cmd)
	if err != nil {
		return err
	}

	switch input.Kind {
	case validateKindCore:
		if err := dict.ValidateCoreEntries(input.Entries); err != nil {
			return err
		}
	case validateKindImport:
		if err := dict.ValidateImportEntries(input.Entries); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported validation kind %q", input.Kind)
	}

	_, err = fmt.Fprint(cmd.OutOrStdout(), formatValidationReport(input.Kind, input.Source, dict.SummarizeEntries(input.Entries)))
	return err
}

type validationInput struct {
	Kind    string
	Source  string
	Entries []dict.Entry
}

func resolveValidationInput(cmd *cobra.Command) (validationInput, error) {
	embeddedCore, err := cmd.Flags().GetBool("embedded-core")
	if err != nil {
		return validationInput{}, fmt.Errorf("get embedded-core flag: %w", err)
	}
	filePath, err := cmd.Flags().GetString("file")
	if err != nil {
		return validationInput{}, fmt.Errorf("get file flag: %w", err)
	}
	filePath = strings.TrimSpace(filePath)

	switch {
	case embeddedCore && filePath != "":
		return validationInput{}, fmt.Errorf("validate cannot use --file with --embedded-core")
	case !embeddedCore && filePath == "":
		return validationInput{}, fmt.Errorf("validate requires either --file or --embedded-core")
	}

	kind, err := resolveValidationKind(cmd, embeddedCore)
	if err != nil {
		return validationInput{}, err
	}

	if embeddedCore {
		entries, err := dict.LoadCoreWords()
		if err != nil {
			return validationInput{}, fmt.Errorf("load embedded core words: %w", err)
		}
		return validationInput{
			Kind:    validateKindCore,
			Source:  "embedded-core",
			Entries: entries,
		}, nil
	}

	format, err := resolveValidationFormat(cmd, filePath)
	if err != nil {
		return validationInput{}, err
	}
	entries, err := loadEntriesForValidation(filePath, format)
	if err != nil {
		return validationInput{}, err
	}
	return validationInput{
		Kind:    kind,
		Source:  filePath,
		Entries: entries,
	}, nil
}

func resolveValidationKind(cmd *cobra.Command, embeddedCore bool) (string, error) {
	flag := cmd.Flags().Lookup("kind")
	kind, err := cmd.Flags().GetString("kind")
	if err != nil {
		return "", fmt.Errorf("get kind flag: %w", err)
	}
	kind = strings.TrimSpace(kind)
	if kind == "" && (flag == nil || !flag.Changed) {
		if embeddedCore {
			return validateKindCore, nil
		}
		return validateKindImport, nil
	}

	switch kind {
	case validateKindCore, validateKindImport:
	default:
		return "", fmt.Errorf("validate only supports --kind %s or %s", validateKindCore, validateKindImport)
	}
	if embeddedCore && kind != validateKindCore {
		return "", fmt.Errorf("validate --embedded-core only supports --kind %s", validateKindCore)
	}
	return kind, nil
}

func resolveValidationFormat(cmd *cobra.Command, filePath string) (string, error) {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return "", fmt.Errorf("get format flag: %w", err)
	}
	format = strings.TrimSpace(strings.ToLower(format))
	if format == "" || format == formatAuto {
		switch strings.ToLower(filepath.Ext(filePath)) {
		case ".csv":
			return formatCSV, nil
		case ".jsonl":
			return formatJSONL, nil
		default:
			return "", fmt.Errorf("infer format from %s: use --format csv or --format jsonl", filePath)
		}
	}

	switch format {
	case formatCSV, formatJSONL:
		return format, nil
	default:
		return "", fmt.Errorf("validate only supports --format %s, %s, or %s", formatAuto, formatCSV, formatJSONL)
	}
}

func loadEntriesForValidation(filePath, format string) ([]dict.Entry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open validation file %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	switch format {
	case formatCSV:
		entries, err := dict.ParseCSV(file)
		if err != nil {
			return nil, fmt.Errorf("parse validation file %s: %w", filePath, err)
		}
		return entries, nil
	case formatJSONL:
		entries, err := dict.ParseJSONL(file)
		if err != nil {
			return nil, fmt.Errorf("parse validation file %s: %w", filePath, err)
		}
		return entries, nil
	default:
		return nil, fmt.Errorf("unsupported validation format %q", format)
	}
}

func formatValidationReport(kind, source string, summary dict.ValidationSummary) string {
	var b strings.Builder
	fmt.Fprintln(&b, "eitango validate")
	fmt.Fprintln(&b, "================")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- kind: %s\n", kind)
	fmt.Fprintf(&b, "- source: %s\n", source)
	fmt.Fprintf(&b, "- entries: %d\n", summary.EntryCount)
	fmt.Fprintf(&b, "- pos: %d\n", summary.PosCount)
	fmt.Fprintf(&b, "- levels: %d\n", summary.LevelCount)
	fmt.Fprintf(&b, "- distractor groups: %d\n", summary.DistractorGroupCount)
	fmt.Fprintf(&b, "- frequency-ranked entries: %d\n", summary.FrequencyRankedEntries)
	return b.String()
}
