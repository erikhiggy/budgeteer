// Package web embeds the built frontend. Run `make build` to populate dist;
// dist/.gitkeep keeps the embed valid before the first build.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

// Dist returns the contents of web/dist.
func Dist() fs.FS {
	dist, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err) // "dist" is a literal that the embed directive guarantees exists
	}
	return dist
}
