package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/dict"
	"github.com/yourname/eitango/internal/store"
)

func newImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import words from CSV",
		Args:  cobra.NoArgs,
		RunE:  runImport,
	}
	cmd.Flags().String("file", "", "CSV file to import")
	cmd.Flags().String("format", "csv", "Import format (csv)")
	cmd.Flags().String("source", "", "Import source name (defaults to the file name)")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func runImport(cmd *cobra.Command, args []string) error {
	if err := validateImportFormat(cmd, "csv"); err != nil {
		return err
	}

	filePath, err := cmd.Flags().GetString("file")
	if err != nil {
		return fmt.Errorf("get file flag: %w", err)
	}
	source, err := importSourceFromFlags(cmd, filePath)
	if err != nil {
		return err
	}

	entries, err := loadImportEntries(filePath)
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

func validateImportFormat(cmd *cobra.Command, expected string) error {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("get format flag: %w", err)
	}
	if format != expected {
		return fmt.Errorf("%s only supports --format %s", cmd.CommandPath(), expected)
	}
	return nil
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

func loadImportEntries(filePath string) ([]dict.Entry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open import file %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	entries, err := dict.ParseCSV(file)
	if err != nil {
		return nil, fmt.Errorf("parse import file %s: %w", filePath, err)
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
