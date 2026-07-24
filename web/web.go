package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist
var distribution embed.FS

func SPAHandler() http.Handler {
	dist, err := fs.Sub(distribution, "dist")
	if err != nil {
		panic(err)
	}

	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestedPath == "." || requestedPath == "" {
			requestedPath = "index.html"
		}

		if _, err := fs.Stat(dist, requestedPath); err == nil {
			files.ServeHTTP(w, r)
			return
		}

		clone := r.Clone(r.Context())
		clone.URL.Path = "/"
		files.ServeHTTP(w, clone)
	})
}
