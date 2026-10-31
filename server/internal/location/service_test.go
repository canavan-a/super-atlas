package location

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

type world struct {
	friends  *friends.Service
	location *Service
	auth     *auth.Store
	hub      *Hub
}

func newWorld(t *testing.T) *world {
	t.Helper()
	conn := testdb.Open(t)
	authStore := auth.NewStore(conn)
	friendStore := friends.NewStore(conn)
	hub := NewHub()
	return &world{
		friends:  friends.NewService(friendStore, authStore, nil),
		location: NewService(NewStore(conn), friendStore, hub),
		auth:     authStore,
		hub:      hub,
	}
}

func (w *world) user(t *testing.T, id, name string) string {
	t.Helper()
	if err := w.auth.CreateUser(context.Background(), id, id+"@example.com", name); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestInviteLinkMakesFriendsWhoSeeEachOther(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	ada := w.user(t, "ada", "Ada")
	bob := w.user(t, "bob", "Bob")

	code, err := w.friends.InviteCode(ctx, ada)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := w.friends.InviteCode(ctx, ada); again != code {
		t.Fatalf("invite code changed between calls: %q then %q", code, again)
	}
	if _, err := w.friends.AcceptInvite(ctx, ada, code); !errors.Is(err, friends.ErrSelfRequest) {
		t.Fatalf("own link: want ErrSelfRequest, got %v", err)
	}
	name, err := w.friends.AcceptInvite(ctx, bob, code)
	if err != nil || name != "Ada" {
		t.Fatalf("accept: %v %q", err, name)
	}
	// Accepting again is a no-op.
	if _, err := w.friends.AcceptInvite(ctx, bob, code); err != nil {
		t.Fatalf("accept twice: %v", err)
	}

	if err := w.location.UpdateLocation(ctx, ada, Fix{Latitude: 1, Longitude: 2, RecordedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	snap, err := w.location.SnapshotForViewer(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap) != 1 || snap[0].UserID != ada || snap[0].DisplayName != "Ada" {
		t.Fatalf("bob's snapshot: %+v", snap)
	}

	// Ghost mode hides Ada.
	if err := w.location.SetPaused(ctx, ada, true); err != nil {
		t.Fatal(err)
	}
	if snap, _ := w.location.SnapshotForViewer(ctx, bob); len(snap) != 0 {
		t.Fatalf("ghost mode: bob still sees %+v", snap)
	}
	list, _ := w.friends.List(ctx, bob)
	if len(list) != 1 || list[0].LastSeenAt != nil {
		t.Fatalf("ghost mode: friends list leaks last seen: %+v", list)
	}
	if !list[0].GhostMode {
		t.Fatalf("ghost mode: bob isn't told ada is a ghost: %+v", list[0])
	}
}

func TestResetInviteRetiresOldLink(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	ada := w.user(t, "ada", "Ada")
	bob := w.user(t, "bob", "Bob")

	old, _ := w.friends.InviteCode(ctx, ada)
	if _, err := w.friends.ResetInviteCode(ctx, ada); err != nil {
		t.Fatal(err)
	}
	if _, err := w.friends.AcceptInvite(ctx, bob, old); !errors.Is(err, friends.ErrInvalidInvite) {
		t.Fatalf("old link: want ErrInvalidInvite, got %v", err)
	}
}

func TestEmailRequestsAndStrangersCantSee(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	ada := w.user(t, "ada", "Ada")
	bob := w.user(t, "bob", "Bob")
	eve := w.user(t, "eve", "Eve")
	_ = w.location.UpdateLocation(ctx, ada, Fix{Latitude: 1, Longitude: 2, RecordedAt: time.Now()})

	f, err := w.friends.SendRequest(ctx, ada, "BOB@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if snap, _ := w.location.SnapshotForViewer(ctx, bob); len(snap) != 0 {
		t.Fatalf("pending request already shares location: %+v", snap)
	}
	// Bob adding Ada back accepts her pending request.
	if back, err := w.friends.SendRequest(ctx, bob, "ada@example.com"); err != nil || back.ID != f.ID || back.Status != "accepted" {
		t.Fatalf("mutual request: %v %+v", err, back)
	}
	if snap, _ := w.location.SnapshotForViewer(ctx, bob); len(snap) != 1 {
		t.Fatalf("accepted friend can't see: %+v", snap)
	}
	if snap, _ := w.location.SnapshotForViewer(ctx, eve); len(snap) != 0 {
		t.Fatalf("stranger sees: %+v", snap)
	}
}

func TestHereSinceAndSpeed(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	ada := w.user(t, "ada", "Ada")
	bob := w.user(t, "bob", "Bob")
	code, _ := w.friends.InviteCode(ctx, ada)
	if _, err := w.friends.AcceptInvite(ctx, bob, code); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	fast := 20.0
	at := func(min int, lat float64, speed *float64) {
		t.Helper()
		if err := w.location.UpdateLocation(ctx, ada, Fix{Latitude: lat, Longitude: 2, SpeedMps: speed, RecordedAt: start.Add(time.Duration(min) * time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(wantSince int, wantSpeed *float64) {
		t.Helper()
		snap, _ := w.location.SnapshotForViewer(ctx, bob)
		if len(snap) != 1 {
			t.Fatalf("snapshot: %+v", snap)
		}
		if want := start.Add(time.Duration(wantSince) * time.Minute).Format(time.RFC3339); snap[0].HereSince != want {
			t.Errorf("here since %s, want %s", snap[0].HereSince, want)
		}
		if (snap[0].SpeedMps == nil) != (wantSpeed == nil) || (wantSpeed != nil && *snap[0].SpeedMps != *wantSpeed) {
			t.Errorf("speed %v, want %v", snap[0].SpeedMps, wantSpeed)
		}
	}
	at(0, 42.65, &fast)
	at(5, 42.70, nil) // 5 km on: arrived somewhere new
	check(5, nil)
	at(20, 42.7004, nil) // ~45 m off: still there
	check(5, nil)
	at(10, 43, nil) // older than what's stored: ignored
	check(5, nil)
	at(30, 42.71, &fast) // ~1 km on: moved again
	check(30, &fast)
}

// TestGhostLeavesOpenMaps: a friend with the map open loses you the moment
// you go into ghost mode, gets nothing while you're a ghost, and gets you
// back when you come out.
func TestGhostLeavesOpenMaps(t *testing.T) {
	ctx := context.Background()
	w := newWorld(t)
	ada := w.user(t, "ada", "Ada")
	bob := w.user(t, "bob", "Bob")
	code, _ := w.friends.InviteCode(ctx, ada)
	if _, err := w.friends.AcceptInvite(ctx, bob, code); err != nil {
		t.Fatal(err)
	}
	bobsMap := &Client{send: make(chan []byte, 8)}
	w.hub.Register(bob, bobsMap)
	next := func() string {
		t.Helper()
		select {
		case p := <-bobsMap.send:
			var m struct{ Type, UserID string }
			_ = json.Unmarshal(p, &m)
			return m.Type + " " + m.UserID
		default:
			return "nothing"
		}
	}
	move := func() {
		_ = w.location.UpdateLocation(ctx, ada, Fix{Latitude: 1, Longitude: 2, RecordedAt: time.Now()})
	}

	move()
	if got := next(); got != "location_update ada" {
		t.Fatalf("before ghost mode bob got %s", got)
	}
	_ = w.location.SetPaused(ctx, ada, true)
	if got := next(); got != "location_removed ada" {
		t.Fatalf("going ghost: bob got %s", got)
	}
	move()
	if got := next(); got != "nothing" {
		t.Fatalf("while a ghost: bob got %s", got)
	}
	_ = w.location.SetPaused(ctx, ada, false)
	if got := next(); got != "location_update ada" {
		t.Fatalf("leaving ghost mode: bob got %s", got)
	}
}
