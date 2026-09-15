package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grisha/serbian-app/server/internal/api"
	"github.com/grisha/serbian-app/server/internal/auth"
	"github.com/grisha/serbian-app/server/internal/config"
	"github.com/grisha/serbian-app/server/internal/content"
	"github.com/grisha/serbian-app/server/internal/mail"
	"github.com/grisha/serbian-app/server/internal/ratelimit"
	"github.com/grisha/serbian-app/server/internal/store"
	"github.com/grisha/serbian-app/server/internal/telegram"
	"github.com/grisha/serbian-app/server/web"
)

func main() {
	loadDotEnv(".env")

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

	cfg := config.Load()

	// Bot /start login/link needs the bot's own @username (not derivable from
	// the token — see Config.TelegramBotID, which only gives the numeric id)
	// and a registered webhook. Both are normally discovered/generated here
	// via a live call to the Bot API — but neither GetMe nor SetWebhook is
	// required to succeed when TELEGRAM_BOT_USERNAME / TELEGRAM_WEBHOOK_SECRET
	// pin them instead (see Config field docs): that lets the feature work
	// even when this host's outbound access to api.telegram.org is blocked,
	// as long as the webhook was registered once from a network that isn't.
	// A failure disables just this one login path (telegram_disabled), the
	// same as an unset TELEGRAM_BOT_TOKEN — never fatal, the way an
	// unreachable SMTP server doesn't stop the process either.
	var telegramBotUsername, telegramWebhookSecret string
	switch {
	case !cfg.TelegramEnabled():
		// disabled entirely — nothing to do
	case !strings.HasPrefix(cfg.AppBaseURL, "https://"):
		log.Printf("telegram: APP_BASE_URL is not https (%s) — webhook needs a public https URL, bot /start login disabled", cfg.AppBaseURL)
	default:
		username := cfg.TelegramBotUsername
		if username == "" {
			u, err := telegram.GetMe(cfg.TelegramBotToken)
			if err != nil {
				log.Printf("telegram: getMe: %v", err)
			} else {
				username = u
			}
		}
		secret := cfg.TelegramWebhookSecret
		if secret == "" {
			s, _, err := auth.NewToken()
			if err != nil {
				log.Printf("telegram: generate webhook secret: %v", err)
			} else {
				secret = s
			}
		}
		if username == "" || secret == "" {
			break // neither discovered nor configured — nothing usable
		}
		webhookURL := cfg.AppBaseURL + "/api/telegram/webhook"
		if err := telegram.SetWebhook(cfg.TelegramBotToken, webhookURL, secret); err != nil {
			log.Printf("telegram: setWebhook: %v (harmless if the webhook was already registered externally)", err)
		}
		telegramBotUsername = username
		telegramWebhookSecret = secret
	}
	sendTelegramMessage := func(chatID int64, text string) {
		if err := telegram.SendMessage(cfg.TelegramBotToken, chatID, text); err != nil {
			log.Printf("telegram: send message: %v", err)
		}
	}

	var mailer mail.Mailer
	if cfg.SMTPEnabled() {
		mailer = mail.NewSMTPMailer(cfg.SMTP)
	} else {
		mailer = mail.LogMailer{}
	}

	sendMail := func(to, subject, text, html string) {
		// Runs inside an Async goroutine already — synchronous here.
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := mailer.Send(ctx, to, subject, text, html); err != nil {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel2()
			if err2 := mailer.Send(ctx2, to, subject, text, html); err2 != nil {
				log.Printf("mail send failed after retry: %v", err2)
			}
		}
	}
	async := func(f func()) { go f() }

	login := ratelimit.NewLimiter(5.0/60, 5)          // 5/min per IP
	loginEmail := ratelimit.NewLimiter(10.0/3600, 10) // 10/hour per email
	slow := ratelimit.NewLimiter(3.0/3600, 3)         // register/resend/forgot: 3/hour
	fails := ratelimit.NewFailCounter(10, 15*time.Minute)

	// Housekeeping: sweep expired sessions hourly.
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for range t.C {
			if _, err := st.DeleteExpiredSessions(time.Now()); err != nil {
				log.Printf("session sweep: %v", err)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/api/", api.Handler(api.Deps{
		Course:                getCourse,
		Store:                 st,
		Now:                   time.Now,
		Stale:                 stale,
		Config:                cfg,
		SendMail:              sendMail,
		Async:                 async,
		Login:                 login,
		LoginEmail:            loginEmail,
		Slow:                  slow,
		Fails:                 fails,
		TelegramBotUsername:   telegramBotUsername,
		TelegramWebhookSecret: telegramWebhookSecret,
		SendTelegramMessage:   sendTelegramMessage,
	}))
	imgDir := filepath.Join(*contentDir, "images")
	mux.Handle("/img/", cacheControl(http.StripPrefix("/img/", http.FileServer(http.Dir(imgDir)))))
	audioDir := filepath.Join(*contentDir, "audio")
	mux.Handle("/audio/", cacheControl(http.StripPrefix("/audio/", http.FileServer(http.Dir(audioDir)))))
	mux.Handle("/", spaHandler(web.FS()))

	log.Printf("listening on %s", *addr)
	// The security headers wrap EVERYTHING, not just /api/: the SPA HTML and the
	// static trees need the CSP too — frame-ancestors is the only clickjacking
	// guard here (there is deliberately no X-Frame-Options, for the Telegram Mini
	// App iframe). api.Handler applies them again inside /api/, which is a
	// harmless no-op overwrite. The origin (CSRF) guard stays scoped to /api/:
	// the SPA and static routes are GET-only, so checking their Origin would buy
	// nothing and could break ordinary cross-site navigation into the app.
	log.Fatal(http.ListenAndServe(*addr, api.SecurityHeaders(cfg, mux)))
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
