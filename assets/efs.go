package assets

import (
	"embed"
	"io/fs"
)

//go:embed "html" "static"
var files embed.FS

var (
	HTMLFiles   = sub(files, "html")
	StaticFiles = sub(files, "static")
)

func sub(f embed.FS, dir string) fs.FS {
	subFS, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return subFS
}
