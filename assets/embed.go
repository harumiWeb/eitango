package assets

import "embed"

// Embedded contains the shipped migrations and core word pack.
//
//go:embed words_core.jsonl migrations/*.sql
var Embedded embed.FS
