package friends

import (
	"context"
	"testing"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

type sent struct{ to, title string }

type fakeNotifier struct{ sent []sent }

func (f *fakeNotifier) Notify(userID, title, body string) {
	f.sent = append(f.sent, sent{userID, title})
}

func TestFriendEventsNotify(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	authStore := auth.NewStore(conn)
	for _, u := range [][2]string{{"ada", "Ada"}, {"bob", "Bob"}, {"cy", "Cy"}} {
		if err := authStore.CreateUser(ctx, u[0], u[0]+"@example.com", u[1]); err != nil {
			t.Fatal(err)
		}
	}
	n := &fakeNotifier{}
	svc := NewService(NewStore(conn), authStore, n)

	f, err := svc.SendRequest(ctx, "ada", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Accept(ctx, f.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	code, _ := svc.InviteCode(ctx, "ada")
	if _, err := svc.AcceptInvite(ctx, "cy", code); err != nil {
		t.Fatal(err)
	}

	want := []sent{
		{"bob", "Ada wants to be friends"},
		{"ada", "Bob is now your friend"},
		{"ada", "Cy is now your friend"},
	}
	if len(n.sent) != len(want) {
		t.Fatalf("sent %+v, want %+v", n.sent, want)
	}
	for i := range want {
		if n.sent[i] != want[i] {
			t.Fatalf("notification %d: got %+v, want %+v", i, n.sent[i], want[i])
		}
	}
}
