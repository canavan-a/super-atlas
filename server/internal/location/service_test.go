package location

import (
	"context"
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
}

func newWorld(t *testing.T) *world {
	t.Helper()
	conn := testdb.Open(t)
	authStore := auth.NewStore(conn)
	friendStore := friends.NewStore(conn)
	return &world{
		friends:  friends.NewService(friendStore, authStore),
		location: NewService(NewStore(conn), friendStore, NewHub()),
		auth:     authStore,
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

	if err := w.location.UpdateLocation(ctx, ada, 1, 2, nil, time.Now()); err != nil {
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
	_ = w.location.UpdateLocation(ctx, ada, 1, 2, nil, time.Now())

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
