package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/canavan-a/super-atlas/server/internal/db"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

type fakeMailer struct{ codes map[string]string }

func (m *fakeMailer) SendLoginCode(to, code string) error {
	m.codes[to] = code
	return nil
}

func newTestService(t *testing.T) (*Service, *fakeMailer, *time.Time) {
	t.Helper()
	mailer := &fakeMailer{codes: map[string]string{}}
	store := NewStore(testdb.Open(t))
	for _, email := range []string{"ada@example.com", "a@b.co"} {
		if err := store.Allow(context.Background(), email); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(store, mailer)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	return svc, mailer, &now
}

func TestSignUpNeedsNameThenSignsIn(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)

	if err := svc.RequestCode(ctx, " Ada@Example.com "); err != nil {
		t.Fatal(err)
	}
	code := mailer.codes["ada@example.com"]

	if _, _, err := svc.Verify(ctx, "ada@example.com", code, ""); !errors.Is(err, ErrNeedsName) {
		t.Fatalf("want ErrNeedsName, got %v", err)
	}
	// The same code still works once a name is supplied.
	user, token, err := svc.Verify(ctx, "ada@example.com", code, "Ada")
	if err != nil {
		t.Fatal(err)
	}
	if user.DisplayName != "Ada" || token == "" {
		t.Fatalf("unexpected user %+v token %q", user, token)
	}
	if id, err := svc.Authenticate(ctx, token); err != nil || id != user.ID {
		t.Fatalf("authenticate: %v %q", err, id)
	}

	// The code can't be reused.
	if _, _, err := svc.Verify(ctx, "ada@example.com", code, "Ada"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("reuse: want ErrInvalidCode, got %v", err)
	}

	// An existing account signs in without a name.
	if err := svc.RequestCode(ctx, "ada@example.com"); err != nil {
		t.Fatal(err)
	}
	again, _, err := svc.Verify(ctx, "ada@example.com", mailer.codes["ada@example.com"], "")
	if err != nil || again.ID != user.ID {
		t.Fatalf("second sign-in: %v %+v", err, again)
	}
}

func TestCodeLocksAfterTooManyWrongGuesses(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)
	_ = svc.RequestCode(ctx, "a@b.co")
	code := mailer.codes["a@b.co"]

	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < maxCodeAttempts; i++ {
		if _, _, err := svc.Verify(ctx, "a@b.co", wrong, "A"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("guess %d: %v", i, err)
		}
	}
	if _, _, err := svc.Verify(ctx, "a@b.co", code, "A"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("right code after lockout: want ErrInvalidCode, got %v", err)
	}
}

func TestCodeExpiresAndNewCodeRetiresOld(t *testing.T) {
	ctx := context.Background()
	svc, mailer, now := newTestService(t)

	_ = svc.RequestCode(ctx, "a@b.co")
	first := mailer.codes["a@b.co"]
	*now = now.Add(time.Second)
	_ = svc.RequestCode(ctx, "a@b.co")
	second := mailer.codes["a@b.co"]
	if first != second {
		if _, _, err := svc.Verify(ctx, "a@b.co", first, "A"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("old code: want ErrInvalidCode, got %v", err)
		}
	}

	*now = now.Add(codeTTL + time.Second)
	if _, _, err := svc.Verify(ctx, "a@b.co", second, "A"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("expired code: want ErrInvalidCode, got %v", err)
	}
}

func signUp(t *testing.T, svc *Service, mailer *fakeMailer, email, name string) string {
	t.Helper()
	ctx := context.Background()
	if err := svc.RequestCode(ctx, email); err != nil {
		t.Fatal(err)
	}
	_, token, err := svc.Verify(ctx, email, mailer.codes[email], name)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestSessionSlidesAndExpiresWhenIdle(t *testing.T) {
	ctx := context.Background()
	svc, mailer, now := newTestService(t)
	token := signUp(t, svc, mailer, "a@b.co", "A")

	// Used every 60 days: never expires.
	for i := 0; i < 5; i++ {
		*now = now.Add(60 * 24 * time.Hour)
		if _, err := svc.Authenticate(ctx, token); err != nil {
			t.Fatalf("after %d active periods: %v", i+1, err)
		}
	}

	// Idle past the TTL: signed out.
	*now = now.Add(sessionIdleTTL + time.Hour)
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("idle session: want ErrInvalidSession, got %v", err)
	}
}

func TestLogoutEndsOnlyThatSession(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)
	phone := signUp(t, svc, mailer, "a@b.co", "A")
	_ = svc.RequestCode(ctx, "a@b.co")
	_, tablet, err := svc.Verify(ctx, "a@b.co", mailer.codes["a@b.co"], "")
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Logout(ctx, phone); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, phone); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("logged-out session: want ErrInvalidSession, got %v", err)
	}
	if _, err := svc.Authenticate(ctx, tablet); err != nil {
		t.Fatalf("other session: %v", err)
	}
}

func TestOnlyAllowedEmailsSignIn(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)

	if err := svc.RequestCode(ctx, "eve@example.com"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("unlisted email: want ErrNotAllowed, got %v", err)
	}
	if _, sent := mailer.codes["eve@example.com"]; sent {
		t.Fatal("a code was sent to an unlisted email")
	}

	// Taking someone off the list signs them out, and a code they already
	// had stops working.
	token := signUp(t, svc, mailer, "a@b.co", "A")
	if err := svc.RequestCode(ctx, "a@b.co"); err != nil {
		t.Fatal(err)
	}
	if listed, err := svc.store.Disallow(ctx, "a@b.co"); err != nil || !listed {
		t.Fatalf("disallow: %v %v", listed, err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("removed user's session: want ErrInvalidSession, got %v", err)
	}
	if _, _, err := svc.Verify(ctx, "a@b.co", mailer.codes["a@b.co"], ""); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("removed user's code: want ErrNotAllowed, got %v", err)
	}

	// Allowing them again restores the same account.
	if err := svc.store.Allow(ctx, "a@b.co"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RequestCode(ctx, "a@b.co"); err != nil {
		t.Fatal(err)
	}
	user, _, err := svc.Verify(ctx, "a@b.co", mailer.codes["a@b.co"], "")
	if err != nil || user.DisplayName != "A" {
		t.Fatalf("re-allowed: %+v %v", user, err)
	}
}

func TestMigrationAllowsExistingAccounts(t *testing.T) {
	// Roll the database back to before the allowlist, make an account, and
	// run the migration again: that account must still be able to sign in.
	ctx := context.Background()
	conn := testdb.Open(t)
	for _, stmt := range []string{
		`DROP TABLE allowed_emails`,
		`DELETE FROM schema_migrations WHERE name = '0007_allowlist.sql'`,
		`INSERT INTO users (id, email, display_name) VALUES ('u1', 'old@example.com', 'Old')`,
	} {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Migrate(ctx, conn); err != nil {
		t.Fatal(err)
	}
	if ok, err := NewStore(conn).IsAllowed(ctx, "old@example.com"); err != nil || !ok {
		t.Fatalf("existing account: %v %v", ok, err)
	}
}
