// Package migrations вшивает SQL-миграции в binary, чтобы накат не зависел
// от того, лежат ли рядом файлы и стоит ли на машине goose CLI.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
