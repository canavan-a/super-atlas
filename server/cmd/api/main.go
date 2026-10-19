package main

import (
	"context"
	"log"
	"net/http"

	"github.com/canavan-a/open-location/server/internal/auth"
	"github.com/canavan-a/open-location/server/internal/config"
	"github.com/canavan-a/open-location/server/internal/db"
	"github.com/canavan-a/open-location/server/internal/friends"
	"github.com/canavan-a/open-location/server/internal/httpapi"
	"github.com/canavan-a/open-location/server/internal/larp"
	"github.com/canavan-a/open-location/server/internal/location"
	"github.com/canavan-a/open-location/server/internal/mail"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Connect(ctx, cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	mailer := mail.NewSender(cfg.SMTPEmail, cfg.SMTPAppPassword)

	authStore := auth.NewStore(conn)
	authService := auth.NewService(authStore, cfg.JWTAccessSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, mailer)
	authHandlers := auth.NewHandlers(authService)

	friendStore := friends.NewStore(conn)
	friendService := friends.NewService(friendStore, authStore)

	hub := location.NewHub()
	locationStore := location.NewStore(conn)
	locationService := location.NewService(locationStore, friendStore, hub)
	locationHandlers := location.NewHandlers(locationService, hub)

	larpStore := larp.NewStore(conn)
	larpService := larp.NewService(larpStore, authStore, friendStore, locationStore, locationService)

	friendHandlers := friends.NewHandlers(friendService, larpService)

	router := httpapi.NewRouter(httpapi.Deps{
		AuthService:      authService,
		AuthHandlers:     authHandlers,
		FriendHandlers:   friendHandlers,
		LocationHandlers: locationHandlers,
	})

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
