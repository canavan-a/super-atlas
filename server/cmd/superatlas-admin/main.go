// Command superatlas-admin manages who's allowed to sign in. Only emails on
// the allowlist get a sign-in code, so nobody else can make an account.
//
//	superatlas-admin allow <email>...   let these emails sign in
//	superatlas-admin remove <email>     take one off and sign it out everywhere
//	superatlas-admin list               everyone allowed, and who's signed up
//
// It works on the database the server uses (SUPERATLAS_DB_PATH or
// SUPERATLAS_DATA_DIR, as for the server), and changes apply immediately.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	case "allow":
		if len(args) == 0 {
			usage()
		}
		for _, arg := range args {
			email := auth.NormalizeEmail(arg)
			if !strings.Contains(email, "@") {
				fail(fmt.Errorf("%q isn't an email", arg))
			}
			if err := store.Allow(ctx, email); err != nil {
				fail(err)
			}
			fmt.Println("allowed", email)
		}
	case "remove":
		if len(args) != 1 {
			usage()
		}
		email := auth.NormalizeEmail(args[0])
		listed, err := store.Disallow(ctx, email)
		if err != nil {
			fail(err)
		}
		if !listed {
			fail(fmt.Errorf("%s isn't on the allowlist", email))
		}
		fmt.Println("removed", email, "(signed out everywhere; their account is kept, allow them again to restore it)")
	case "list":
		allowed, err := store.ListAllowed(ctx)
		if err != nil {
			fail(err)
		}
		if len(allowed) == 0 {
			fmt.Println("nobody is allowed yet: superatlas-admin allow <email>")
		}
		for _, a := range allowed {
			if a.DisplayName != "" {
				fmt.Printf("%s\t%s\n", a.Email, a.DisplayName)
			} else {
				fmt.Printf("%s\t(hasn't signed up)\n", a.Email)
			}
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: superatlas-admin allow <email>... | remove <email> | list")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "superatlas-admin:", err)
	os.Exit(1)
}
