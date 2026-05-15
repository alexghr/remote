package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func New() http.Handler {
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	return http.FileServer(http.FS(files))
}
