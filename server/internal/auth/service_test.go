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

func TestReviewAccountSignsInWithFixedCode(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)
	svc.SetReviewAccount("Tester@Example.com", "222222")

	// Not on the allowlist, and nothing is emailed.
	if err := svc.RequestCode(ctx, "tester@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, sent := mailer.codes["tester@example.com"]; sent {
		t.Fatal("a code was emailed to the review account")
	}

	if _, _, err := svc.Verify(ctx, "tester@example.com", "123456", "Tester"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("wrong code: want ErrInvalidCode, got %v", err)
	}
	if _, _, err := svc.Verify(ctx, "tester@example.com", "222222", ""); !errors.Is(err, ErrNeedsName) {
		t.Fatalf("new account: want ErrNeedsName, got %v", err)
	}
	first, _, err := svc.Verify(ctx, "tester@example.com", "222222", "Tester")
	if err != nil {
		t.Fatal(err)
	}
	// The code keeps working, on the same account, without asking again.
	again, token, err := svc.Verify(ctx, "tester@example.com", "222222", "")
	if err != nil || again.ID != first.ID {
		t.Fatalf("second sign-in: %+v %v", again, err)
	}
	if id, err := svc.Authenticate(ctx, token); err != nil || id != first.ID {
		t.Fatalf("authenticate: %v %q", err, id)
	}

	// Everyone else still needs a real code.
	if _, _, err := svc.Verify(ctx, "ada@example.com", "222222", "Ada"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("other email with the review code: want ErrInvalidCode, got %v", err)
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

func TestDeleteAccountRemovesEverything(t *testing.T) {
	ctx := context.Background()
	svc, mailer, _ := newTestService(t)
	signUp := func(email, name string) (*User, string) {
		t.Helper()
		if err := svc.RequestCode(ctx, email); err != nil {
			t.Fatal(err)
		}
		user, token, err := svc.Verify(ctx, email, mailer.codes[email], name)
		if err != nil {
			t.Fatal(err)
		}
		return user, token
	}
	ada, token := signUp("ada@example.com", "Ada")
	bob, _ := signUp("a@b.co", "Bob")

	conn := svc.store.db
	for _, q := range []string{
		`INSERT INTO friendships (id, requester_id, addressee_id, status) VALUES ('f1', ?1, ?2, 'accepted')`,
		`INSERT INTO locations (user_id, latitude, longitude, recorded_at) VALUES (?1, 1, 2, '2026-01-01T00:00:00Z')`,
		`INSERT INTO user_settings (user_id, sharing_paused) VALUES (?1, 1)`,
		`INSERT INTO push_tokens (token, user_id, session_id) SELECT 'tok', ?1, id FROM sessions WHERE user_id = ?1`,
	} {
		if _, err := conn.ExecContext(ctx, q, ada.ID, bob.ID); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	// A code requested and never used.
	if err := svc.RequestCode(ctx, "ada@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteAccount(ctx, ada.ID); err != nil {
		t.Fatal(err)
	}

	for table, where := range map[string]string{
		"users":          "id = ?1",
		"sessions":       "user_id = ?1",
		"friendships":    "requester_id = ?1 OR addressee_id = ?1",
		"locations":      "user_id = ?1",
		"user_settings":  "user_id = ?1",
		"push_tokens":    "user_id = ?1",
		"login_codes":    "email = ?2",
		"allowed_emails": "email = ?2",
	} {
		var n int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+where, ada.ID, "ada@example.com").Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d rows left", table, n)
		}
	}
	if _, err := svc.Authenticate(ctx, token); err == nil {
		t.Error("the deleted account's session still works")
	}
	// Nobody else is touched, and the email can't sign up again until allowed.
	if _, err := svc.GetUser(ctx, bob.ID); err != nil {
		t.Errorf("other user: %v", err)
	}
	if err := svc.RequestCode(ctx, "ada@example.com"); !errors.Is(err, ErrNotAllowed) {
		t.Errorf("deleted email: want ErrNotAllowed, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, ada.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleting twice: want ErrNotFound, got %v", err)
	}
}
