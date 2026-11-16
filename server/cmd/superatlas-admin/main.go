// Command superatlas-admin manages the cap on accounts. Anyone can sign up
// until there are that many (150 to start); after that only people who
// already have an account can sign in.
//
//	superatlas-admin cap              how many accounts there are, and the cap
//	superatlas-admin cap <n>          set the cap (raise it to let more in)
//	superatlas-admin list             everyone with an account, oldest first
//	superatlas-admin remove <email>   delete that account and all its data
//
// It works on the database the server uses (SUPERATLAS_DB_PATH or
// SUPERATLAS_DATA_DIR, as for the server), and changes apply immediately.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/config"
	"github.com/canavan-a/super-atlas/server/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg := config.Load()
	ctx := context.Background()

	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o750); err != nil {
		fail(err)
	}
	conn, err := db.Connect(ctx, cfg.DatabasePath)
	if err != nil {
		fail(err)
	}
	defer conn.Close()
	// So it works before the server has ever run.
	if err := db.Migrate(ctx, conn); err != nil {
		fail(err)
	}
	store := auth.NewStore(conn)

	args := os.Args[2:]
	switch os.Args[1] {
	case "cap":
		switch len(args) {
		case 0:
		case 1:
			n, err := strconv.Atoi(args[0])
			if err != nil || n < 0 {
				fail(fmt.Errorf("%q isn't a number of accounts", args[0]))
			}
			if err := store.SetUserCap(ctx, n); err != nil {
				fail(err)
			}
		default:
			usage()
		}
		count, err := store.CountUsers(ctx)
		if err != nil {
			fail(err)
		}
		limit, err := store.UserCap(ctx)
		if err != nil {
			fail(err)
		}
		fmt.Printf("%d of %d accounts", count, limit)
		if count >= limit {
			fmt.Print(" (full: nobody new can sign up)")
		}
		fmt.Println()
	case "list":
		if len(args) != 0 {
			usage()
		}
		users, err := store.ListUsers(ctx)
		if err != nil {
			fail(err)
		}
		if len(users) == 0 {
			fmt.Println("nobody has signed up yet")
		}
		for _, u := range users {
			fmt.Printf("%s\t%s\t%s\n", u.CreatedAt.Format("2006-01-02"), u.Email, u.DisplayName)
		}
	case "remove":
		if len(args) != 1 {
			usage()
		}
		email := auth.NormalizeEmail(args[0])
		user, err := store.GetUserByEmail(ctx, email)
		if errors.Is(err, auth.ErrNotFound) {
			fail(fmt.Errorf("%s has no account", email))
		}
		if err != nil {
			fail(err)
		}
		if err := store.DeleteUser(ctx, user.ID); err != nil {
			fail(err)
		}
		fmt.Println("deleted", email, "(signed out everywhere, friends and location gone; they can sign up again while there's room)")
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: superatlas-admin cap [<n>] | list | remove <email>")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "superatlas-admin:", err)
	os.Exit(1)
}
