// Package avatar keeps profile pictures. The app uploads a photo; the server
// decodes it, crops it to the middle square and re-encodes it as a small
// JPEG, so what it stores and serves never carries the original's metadata
// (camera EXIF, GPS). Without one, the app draws the user's initials.
package avatar

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"

	"github.com/canavan-a/super-atlas/server/internal/auth"
	"github.com/go-chi/chi/v5"
)

const (
	// Size is the stored picture's width and height.
	Size = 256
	// maxUpload is the largest photo accepted. The app sends a cropped,
	// compressed one, well under this.
	maxUpload = 8 << 20
	// maxPixels refuses images whose header claims more pixels than any phone
	// photo, before decoding them.
	maxPixels = 50_000_000
)

var ErrBadImage = errors.New("not a JPEG or PNG image")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Set replaces userID's picture, returning its new id.
func (s *Store) Set(ctx context.Context, userID string, jpegData []byte) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b[:])
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO avatars (user_id, id, jpeg) VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET id = excluded.id, jpeg = excluded.jpeg,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
	`, userID, id, jpegData)
	return id, err
}

func (s *Store) Delete(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM avatars WHERE user_id = ?`, userID)
	return err
}

// Get returns the picture with this id, or nil if there's none.
func (s *Store) Get(ctx context.Context, id string) ([]byte, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT jpeg FROM avatars WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

// Process decodes a JPEG or PNG, crops it to its middle square, scales that
// to Size x Size and encodes it as a JPEG.
func Process(data []byte) ([]byte, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width*cfg.Height > maxPixels {
		return nil, ErrBadImage
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrBadImage
	}
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	if side == 0 {
		return nil, ErrBadImage
	}
	x0 := b.Min.X + (b.Dx()-side)/2
	y0 := b.Min.Y + (b.Dy()-side)/2
	out := image.NewRGBA(image.Rect(0, 0, Size, Size))
	// Box filter: each output pixel averages the source pixels it covers
	// (one, when the source is smaller than Size).
	for y := range Size {
		sy0, sy1 := y0+y*side/Size, y0+(y+1)*side/Size
		sy1 = max(sy1, sy0+1)
		for x := range Size {
			sx0, sx1 := x0+x*side/Size, x0+(x+1)*side/Size
			sx1 = max(sx1, sx0+1)
			var r, g, bl, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, _ := src.At(sx, sy).RGBA()
					r, g, bl, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), n+1
				}
			}
			out.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), 0xffff})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Handlers struct {
	store *Store
	// changed is told whose picture changed, so caches of it (the location
	// service's, for live pushes) can drop it and friends' maps update.
	changed func(ctx context.Context, userID string)
}

func NewHandlers(store *Store, changed func(ctx context.Context, userID string)) *Handlers {
	return &Handlers{store: store, changed: changed}
}

// Put sets the caller's picture from the request body, a JPEG or PNG.
func (h *Handlers) Put(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxUpload))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "that photo is too big")
		return
	}
	processed, err := Process(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "that photo couldn't be read, try a JPEG or PNG")
		return
	}
	id, err := h.store.Set(r.Context(), userID, processed)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save photo")
		return
	}
	h.changed(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]string{"avatarId": id})
}

// Delete removes the caller's picture.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.store.Delete(r.Context(), userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove photo")
		return
	}
	h.changed(r.Context(), userID)
	w.WriteHeader(http.StatusNoContent)
}

// Serve returns a picture by its id. It needs no sign-in, so the map and an
// <img> can load it directly: the id is random, only the user's friends and
// people they have friend requests with are given it, and it changes when
// the picture does.
func (h *Handlers) Serve(w http.ResponseWriter, r *http.Request) {
	data, err := h.store.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "could not load photo", http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
