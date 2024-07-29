// Package docs exports the embedded markdown files in the docs directory.
package docs

import (
	"embed"
	_ "embed"
)

//go:embed *.md
var All embed.FS
