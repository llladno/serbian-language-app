package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/api"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/web"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	contentDir := flag.String("content", "./content", "content directory")
	dbPath := flag.String("db", "./data/app.db", "SQLite path (used when DATABASE_URL is unset)")
	dsnFlag := flag.String("dsn", "", "database DSN; a postgres:// URL selects Postgres (overrides -db; env DATABASE_URL wins if set)")
	importSQLite := flag.String("import-sqlite", "", "one-time: copy rows from this SQLite file into the target DB if it has no accounts, then continue")
	flag.Parse()

	dsn := *dsnFlag
	if env := os.Getenv("DATABASE_URL"); env != "" {
		dsn = env
	}
	if dsn == "" {
		dsn = *dbPath
	}

	getCourse, stale, err := content.Watch(*contentDir)
	if err != nil {
		log.Fatalf("load content: %v", err)
	}

	if !store.IsPostgresDSN(dsn) {
		if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
			log.Fatalf("data dir: %v", err)
		}
	}
	st, err := store.Open(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer st.Close()

	if *importSQLite != "" {
		n, err := store.ImportSQLite(st, *importSQLite)
		if err != nil {
			log.Fatalf("import from %s: %v", *importSQLite, err)
		}
		log.Printf("imported %d rows from %s", n, *importSQLite)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(api.Deps{
		Course: getCourse,
		Store:  st,
		Now:    time.Now,
		Stale:  stale,
	}))
	imgDir := filepath.Join(*contentDir, "images")
	mux.Handle("/img/", cacheControl(http.StripPrefix("/img/", http.FileServer(http.Dir(imgDir)))))
	audioDir := filepath.Join(*contentDir, "audio")
	mux.Handle("/audio/", cacheControl(http.StripPrefix("/audio/", http.FileServer(http.Dir(audioDir)))))
	mux.Handle("/", spaHandler(web.FS()))

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func cacheControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		h.ServeHTTP(w, r)
	})
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
