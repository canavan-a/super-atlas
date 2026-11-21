package friends

import (
	"context"
	"testing"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

// Hiding a friend is one-sided: only the one who hid them sees hiddenOnMap.
func TestHideFromMap(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	authStore := auth.NewStore(conn)
	for _, u := range [][2]string{{"ada", "Ada"}, {"bob", "Bob"}} {
		if err := authStore.CreateUser(ctx, u[0], u[0]+"@example.com", u[1]); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(NewStore(conn), authStore, &fakeNotifier{})
	f, err := svc.SendRequest(ctx, "ada", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SetHidden(ctx, f.ID, "bob", true); err != ErrNotFound {
		t.Fatalf("hiding a pending request: got %v, want ErrNotFound", err)
	}
	if err := svc.Accept(ctx, f.ID, "bob"); err != nil {
		t.Fatal(err)
	}

	hidden := func(user string) bool {
		views, err := svc.List(ctx, user)
		if err != nil || len(views) != 1 {
			t.Fatalf("list %s: %v %+v", user, err, views)
		}
		return views[0].HiddenOnMap
	}
	for _, step := range []struct {
		user   string
		hidden bool
		ada    bool
		bob    bool
	}{
		{"bob", true, false, true},
		{"ada", true, true, true},
		{"bob", false, true, false},
	} {
		if err := svc.SetHidden(ctx, f.ID, step.user, step.hidden); err != nil {
			t.Fatal(err)
		}
		if hidden("ada") != step.ada || hidden("bob") != step.bob {
			t.Fatalf("after %s hidden=%v: ada=%v bob=%v", step.user, step.hidden, hidden("ada"), hidden("bob"))
		}
	}
	if err := svc.SetHidden(ctx, f.ID, "cy", true); err != ErrNotYourRequest {
		t.Fatalf("stranger: got %v", err)
	}
}
