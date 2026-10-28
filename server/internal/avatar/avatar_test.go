package avatar

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/canavan-a/super-atlas/server/internal/testdb"
)

// A 600x300 PNG, red on the left third, green in the middle, blue on the
// right: the middle square is all green.
func stripes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 600, 300))
	for y := range 300 {
		for x := range 600 {
			c := color.RGBA{255, 0, 0, 255}
			if x >= 150 && x < 450 {
				c = color.RGBA{0, 255, 0, 255}
			} else if x >= 450 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessCropsMiddleSquareToJPEG(t *testing.T) {
	out, err := Process(stripes(t))
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("not a JPEG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != Size || b.Dy() != Size {
		t.Fatalf("size %v, want %dx%d", b, Size, Size)
	}
	for _, p := range []image.Point{{2, 2}, {Size / 2, Size / 2}, {Size - 3, Size - 3}} {
		r, g, b, _ := img.At(p.X, p.Y).RGBA()
		if g>>8 < 200 || r>>8 > 60 || b>>8 > 60 {
			t.Errorf("pixel %v is %d,%d,%d, want green", p, r>>8, g>>8, b>>8)
		}
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	if _, err := Process([]byte("not an image")); err != ErrBadImage {
		t.Fatalf("err %v, want ErrBadImage", err)
	}
}

func TestPictureFollowsUserAndGoesWithAccount(t *testing.T) {
	ctx := context.Background()
	conn := testdb.Open(t)
	users := auth.NewStore(conn)
	if err := users.CreateUser(ctx, "u1", "u1@example.com", "Aidan Canavan"); err != nil {
		t.Fatal(err)
	}
	store := NewStore(conn)
	data, err := Process(stripes(t))
	if err != nil {
		t.Fatal(err)
	}

	first, err := store.Set(ctx, "u1", data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Set(ctx, "u1", data)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("a new picture kept the old id")
	}
	if got, _ := store.Get(ctx, first); got != nil {
		t.Fatal("the old id still serves a picture")
	}
	u, err := users.GetUserByID(ctx, "u1")
	if err != nil || u.AvatarID != second {
		t.Fatalf("user avatar %q (%v), want %q", u.AvatarID, err, second)
	}

	if err := users.DeleteUser(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.Get(ctx, second); got != nil {
		t.Fatal("picture survived deleting the account")
	}
}
