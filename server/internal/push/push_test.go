package push

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendAndUnregistered(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		msg := got["message"].(map[string]any)
		if msg["token"] == "gone" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"name":"projects/x/messages/1"}`))
	}))
	defer srv.Close()
	f := NewWithEndpoint(srv.URL, srv.Client())

	if err := f.Send(context.Background(), "tok", "Ada wants to be friends", "Open Super Atlas to accept."); err != nil {
		t.Fatal(err)
	}
	msg := got["message"].(map[string]any)
	n := msg["notification"].(map[string]any)
	if msg["token"] != "tok" || n["title"] != "Ada wants to be friends" {
		t.Fatalf("unexpected message %v", got)
	}

	if err := f.Send(context.Background(), "gone", "t", "b"); !errors.Is(err, ErrUnregistered) {
		t.Fatalf("want ErrUnregistered, got %v", err)
	}

	var nilFCM *FCM
	if err := nilFCM.Send(context.Background(), "tok", "t", "b"); err != nil {
		t.Fatalf("nil FCM should send nothing, got %v", err)
	}
}
