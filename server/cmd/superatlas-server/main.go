package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/avatar"
	"github.com/canavan-a/super-atlas/server/internal/config"
	"github.com/canavan-a/super-atlas/server/internal/db"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/httpapi"
	"github.com/canavan-a/super-atlas/server/internal/larp"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/canavan-a/super-atlas/server/internal/mail"
	"github.com/canavan-a/super-atlas/server/internal/notify"
	"github.com/canavan-a/super-atlas/server/internal/npc"
	"github.com/canavan-a/super-atlas/server/internal/push"
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

	mailer := mail.NewSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPEmail, cfg.SMTPAppPassword, cfg.Dev)
	if (cfg.SMTPEmail == "" || cfg.SMTPAppPassword == "") && !cfg.Dev {
		log.Printf("warning: SMTP isn't configured, so nobody can sign in")
	}

	authStore := auth.NewStore(conn)
	authService := auth.NewService(authStore, mailer)
	if cfg.ReviewEmail != "" && cfg.ReviewCode != "" {
		authService.SetReviewAccount(cfg.ReviewEmail, cfg.ReviewCode)
		log.Printf("review account on: %s signs in with its fixed code", cfg.ReviewEmail)
	}
	go authService.RunJanitor(ctx)

	// Friend notifications: the sideload app's background connection, plus
	// FCM for the Google Play build when a Firebase key is configured.
	var fcm *push.FCM
	if cfg.FCMCredentialsFile != "" {
		key, err := os.ReadFile(cfg.FCMCredentialsFile)
		if err != nil {
			log.Fatalf("reading SUPERATLAS_FCM_CREDENTIALS_FILE: %v", err)
		}
		if fcm, err = push.New(key); err != nil {
			log.Fatalf("FCM: %v", err)
		}
		log.Printf("push: FCM notifications on")
	}
	notifyService := notify.NewService(notify.NewStore(conn), fcm)

	friendStore := friends.NewStore(conn)
	friendService := friends.NewService(friendStore, authStore, notifyService)

	hub := location.NewHub()
	locationStore := location.NewStore(conn)
	locationService := location.NewService(locationStore, friendStore, hub)
	locationHandlers := location.NewHandlers(locationService, hub)

	// Simulated friends around Albany for testing: a friend request to one
	// of their @lunargen.com emails is accepted at once and they start
	// driving on your map.
	npcs := npc.NewService(authStore, friendStore, locationService)
	if err := npcs.Setup(ctx); err != nil {
		log.Fatalf("npc: %v", err)
	}
	friendService.SetAutoAccept(npcs.IsNPC)
	go npcs.Run(ctx)

	// The "larp" fake-friends trigger only exists in dev mode.
	var larpToggler friends.LarpToggler
	if cfg.Dev {
		larpToggler = larp.NewService(larp.NewStore(conn), authStore, friendStore, locationStore, locationService)
	}

	router := httpapi.NewRouter(httpapi.Deps{
		Config:           cfg,
		AuthService:      authService,
		AuthHandlers:     auth.NewHandlers(authService, locationService),
		AvatarHandlers:   avatar.NewHandlers(avatar.NewStore(conn), locationService.ProfileChanged),
		FriendService:    friendService,
		FriendHandlers:   friends.NewHandlers(friendService, larpToggler, httpapi.InviteURL(cfg)),
		LocationHandlers: locationHandlers,
		NotifyHandlers:   notify.NewHandlers(notifyService),
	})

	log.Printf("superatlas listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatal(err)
	}
}
