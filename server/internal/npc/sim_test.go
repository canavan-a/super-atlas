package npc

import (
	"context"
	"testing"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/friends"
	"github.com/canavan-a/super-atlas/server/internal/location"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

type countingNotifier struct{ n int }

func (c *countingNotifier) Notify(userID, title, body string) { c.n++ }

func TestRequestToNPCStartsSharing(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	authStore := auth.NewStore(conn)
	friendStore := friends.NewStore(conn)
	notifier := &countingNotifier{}
	friendService := friends.NewService(friendStore, authStore, notifier)
	locationStore := location.NewStore(conn)
	locationService := location.NewService(locationStore, friendStore, location.NewHub())
	if err := authStore.CreateUser(ctx, "ada", "ada@example.com", "Ada"); err != nil {
		t.Fatal(err)
	}

	sim := NewService(authStore, friendStore, locationService)
	if err := sim.Setup(ctx); err != nil {
		t.Fatal(err)
	}
	// A second start finds the same accounts rather than making new ones.
	if err := sim.Setup(ctx); err != nil {
		t.Fatal(err)
	}
	if len(sim.userIDs) != len(cast) {
		t.Fatalf("%d NPC accounts, want %d", len(sim.userIDs), len(cast))
	}
	friendService.SetAutoAccept(sim.IsNPC)

	f, err := friendService.SendRequest(ctx, "ada", "Nadia@lunargen.com")
	if err != nil {
		t.Fatal(err)
	}
	if f.Status != "accepted" || notifier.n != 0 {
		t.Fatalf("status %q with %d notifications, want accepted with none", f.Status, notifier.n)
	}

	runCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()
	sim.Run(runCtx)
	snap, err := locationService.SnapshotForViewer(ctx, "ada")
	if err != nil {
		t.Fatal(err)
	}
	if len(snap) != 1 || snap[0].DisplayName != "Nadia Brennan" {
		t.Fatalf("ada sees %+v, want only Nadia", snap)
	}
	// Nobody friended the others, so they send nothing.
	ids := make([]string, 0, len(sim.userIDs))
	for id := range sim.userIDs {
		ids = append(ids, id)
	}
	if stored, _ := locationStore.GetForUsers(ctx, ids); len(stored) != 1 || stored[0].UserID != f.AddresseeID {
		t.Fatalf("NPC positions stored: %+v, want only Nadia's", stored)
	}
}
