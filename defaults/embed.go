// Package defaults holds gh-prs's built-in config, commands and prompts, embedded in every front end.
package defaults

import "embed"

//go:embed config.yml commands prompts
var FS embed.FS
