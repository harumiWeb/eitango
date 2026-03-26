package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/dict"
	"github.com/yourname/eitango/internal/store"
)

func newImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import words from CSV or JSONL",
		Args:  cobra.NoArgs,
		RunE:  runImport,
	}
	cmd.Flags().String("file", "", "Dictionary file to import (.csv or .jsonl)")
	cmd.Flags().String("format", formatCSV, "Import format (csv or jsonl)")
	cmd.Flags().String("source", "", "Import source name (defaults to the file name)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	filePath, err := cmd.Flags().GetString("file")
	if err != nil {
		return fmt.Errorf("get file flag: %w", err)
	}
	format, err := resolveImportFormat(cmd)
	if err != nil {
		return err
	}
	source, err := importSourceFromFlags(cmd, filePath)
	if err != nil {
		return err
	}

	entries, err := loadImportEntries(filePath, format)
	if err != nil {
		return err
	}

	ctx := commandContext(cmd)
	st, _, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	result, err := st.ImportWords(ctx, source, entries)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(cmd.OutOrStdout(), formatImportReport(result))
	return err
}

func resolveImportFormat(cmd *cobra.Command) (string, error) {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return "", fmt.Errorf("get format flag: %w", err)
	}
	normalized := strings.TrimSpace(strings.ToLower(format))
	switch normalized {
	case formatCSV, formatJSONL:
		return normalized, nil
	default:
		return "", fmt.Errorf("%s only supports --format %s or %s", cmd.CommandPath(), formatCSV, formatJSONL)
	}
}

func importSourceFromFlags(cmd *cobra.Command, filePath string) (string, error) {
	sourceName, err := cmd.Flags().GetString("source")
	if err != nil {
		return "", fmt.Errorf("get source flag: %w", err)
	}
	if strings.TrimSpace(sourceName) != "" {
		return store.NormalizeImportSource(sourceName)
	}
	return store.DefaultImportSource(filePath)
}

func loadImportEntries(filePath, format string) ([]dict.Entry, error) {
	entries, err := loadEntriesForValidation(filePath, format)
	if err != nil {
		return nil, fmt.Errorf("load import file %s: %w", filePath, err)
	}
	if err := dict.ValidateImportEntries(entries); err != nil {
		return nil, fmt.Errorf("validate import file %s: %w", filePath, err)
	}
	return entries, nil
}

func formatImportReport(result store.ImportResult) string {
	var b strings.Builder
	fmt.Fprintln(&b, "eitango import")
	fmt.Fprintln(&b, "==============")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- source: %s\n", result.Source)
	fmt.Fprintf(&b, "- inserted words: %d\n", result.InsertedWords)
	fmt.Fprintf(&b, "- updated words: %d\n", result.UpdatedWords)
	return b.String()
}
