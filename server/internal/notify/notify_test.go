package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

type nopMailer struct{ code string }

func (m *nopMailer) SendLoginCode(_, code string) error {
	m.code = code
	return nil
}

func TestPushTokenGoesWithItsSession(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	mailer := &nopMailer{}
	authSvc := auth.NewService(auth.NewStore(conn), mailer)
	if err := authSvc.RequestCode(ctx, "ada@example.com"); err != nil {
		t.Fatal(err)
	}
	user, token, err := authSvc.Verify(ctx, "ada@example.com", mailer.code, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	var sessionID string
	if err := conn.QueryRow(`SELECT id FROM sessions WHERE user_id = ?`, user.ID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}

	store := NewStore(conn)
	if err := store.SetToken(ctx, "device-1", user.ID, sessionID); err != nil {
		t.Fatal(err)
	}
	if tokens, _ := store.TokensFor(ctx, user.ID); len(tokens) != 1 {
		t.Fatalf("tokens before logout: %v", tokens)
	}

	if err := authSvc.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if tokens, _ := store.TokensFor(ctx, user.ID); len(tokens) != 0 {
		t.Fatalf("tokens after logout: %v", tokens)
	}
}

func TestNotifyReachesTheStream(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	mailer := &nopMailer{}
	authSvc := auth.NewService(auth.NewStore(conn), mailer)
	_ = authSvc.RequestCode(ctx, "ada@example.com")
	user, token, err := authSvc.Verify(ctx, "ada@example.com", mailer.code, "Ada")
	if err != nil {
		t.Fatal(err)
	}

	svc := NewService(NewStore(conn), nil)
	srv := httptest.NewServer(authSvc.RequireAuth(http.HandlerFunc(NewHandlers(svc).Stream)))
	defer srv.Close()

	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"?token="+token, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()

	// The socket registers after the upgrade; retry until it's there.
	deadline := time.Now().Add(2 * time.Second)
	_ = ws.SetReadDeadline(deadline)
	got := make(chan Event, 1)
	go func() {
		var e Event
		if ws.ReadJSON(&e) == nil {
			got <- e
		}
	}()
	for time.Now().Before(deadline) {
		svc.Notify(user.ID, "Bob is now your friend", "You can see each other on the map.")
		select {
		case e := <-got:
			if e.Type != "notify" || e.Title != "Bob is now your friend" {
				t.Fatalf("unexpected event %+v", e)
			}
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatal("no notification arrived")
}

func TestStreamRejectsSignedOut(t *testing.T) {
	conn := testdb.Open(t)
	authSvc := auth.NewService(auth.NewStore(conn), &nopMailer{})
	svc := NewService(NewStore(conn), nil)
	srv := httptest.NewServer(authSvc.RequireAuth(http.HandlerFunc(NewHandlers(svc).Stream)))
	defer srv.Close()

	_, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"?token=nope", nil)
	if err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %v %v", resp, err)
	}
}
