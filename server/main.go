package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/grisha/serbian-app/server/internal/api"
	"github.com/grisha/serbian-app/server/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	_ = flag.String("content", "./content", "content directory")
	_ = flag.String("db", "./data/app.db", "sqlite path")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(api.Deps{}))
	mux.Handle("/", spaHandler(web.FS()))

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// spaHandler serves static files and falls back to index.html for
// client-side routes (paths without a file extension).
func spaHandler(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(files, p); err != nil {
			if !strings.Contains(p, ".") {
				r = r.Clone(r.Context())
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
