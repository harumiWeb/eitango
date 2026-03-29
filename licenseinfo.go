package eitango

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed LICENSE THIRD_PARTY_NOTICES.md
var embeddedLicenses embed.FS

type embeddedLicenseDocument struct {
	path  string
	title string
}

var embeddedLicenseDocuments = []embeddedLicenseDocument{
	{path: "LICENSE", title: "LICENSE"},
	{path: "THIRD_PARTY_NOTICES.md", title: "THIRD_PARTY_NOTICES.md"},
}

func LicenseText() (string, error) {
	var sections []string
	for _, document := range embeddedLicenseDocuments {
		body, err := embeddedLicenses.ReadFile(document.path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", document.path, err)
		}
		sections = append(sections, formatEmbeddedLicenseSection(document.title, string(body)))
	}
	return strings.Join(sections, "\n\n"), nil
}

func formatEmbeddedLicenseSection(title, body string) string {
	body = strings.TrimRight(body, "\r\n")
	return fmt.Sprintf("===== %s =====\n\n%s", title, body)
}
