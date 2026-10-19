package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/config"
	"github.com/canavan-a/super-atlas/server/internal/db"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/httpapi"
	"github.com/canavan-a/super-atlas/server/internal/larp"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/canavan-a/super-atlas/server/internal/mail"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o750); err != nil {
		log.Fatalf("data dir: %v", err)
	}
	conn, err := db.Connect(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	mailer := mail.NewSender(cfg.SMTPEmail, cfg.SMTPAppPassword, cfg.Dev)
	if (cfg.SMTPEmail == "" || cfg.SMTPAppPassword == "") && !cfg.Dev {
		log.Printf("warning: SMTP isn't configured, so nobody can sign in")
	}

	authStore := auth.NewStore(conn)
	authService := auth.NewService(authStore, mailer)
	go authService.RunJanitor(ctx)

	friendStore := friends.NewStore(conn)
	friendService := friends.NewService(friendStore, authStore)

	hub := location.NewHub()
	locationStore := location.NewStore(conn)
	locationService := location.NewService(locationStore, friendStore, hub)
	locationHandlers := location.NewHandlers(locationService, hub)

	// The "larp" fake-friends trigger only exists in dev mode.
	var larpToggler friends.LarpToggler
	if cfg.Dev {
		larpToggler = larp.NewService(larp.NewStore(conn), authStore, friendStore, locationStore, locationService)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Config:           cfg,
		AuthService:      authService,
		AuthHandlers:     auth.NewHandlers(authService, locationService),
		FriendService:    friendService,
		FriendHandlers:   friends.NewHandlers(friendService, larpToggler, httpapi.InviteURL(cfg)),
		LocationHandlers: locationHandlers,
	})

	log.Printf("superatlas listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatal(err)
	}
}
