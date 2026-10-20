// Package push sends notifications to the Google Play build of the app
// through Firebase Cloud Messaging (app/src/notifications.ts registers each
// device's token). Adapted from super-chat's server/push, which wakes room
// topics; here the server knows who's friends with whom anyway, so it sends a
// readable notification straight to the user's devices.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ErrUnregistered means the token is no longer valid (the app was
// uninstalled, or its data cleared): the caller should forget it.
var ErrUnregistered = errors.New("FCM token unregistered")

// FCM sends notifications. A nil *FCM is valid and sends nothing.
type FCM struct {
	client *http.Client
	// endpoint is FCM's send URL; tests point it elsewhere.
	endpoint string
}

// New reads a Firebase service account key (Project settings → Service
// accounts → Generate new private key).
func New(credentialsJSON []byte) (*FCM, error) {
	creds, err := google.CredentialsFromJSON(context.Background(), credentialsJSON,
		"https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return nil, fmt.Errorf("FCM credentials: %w", err)
	}
	if creds.ProjectID == "" {
		return nil, fmt.Errorf("FCM credentials: no project_id")
	}
	return NewWithEndpoint(
		"https://fcm.googleapis.com/v1/projects/"+creds.ProjectID+"/messages:send",
		oauth2.NewClient(context.Background(), creds.TokenSource),
	), nil
}

// NewWithEndpoint sends to endpoint instead of FCM (tests).
func NewWithEndpoint(endpoint string, client *http.Client) *FCM {
	client.Timeout = 10 * time.Second
	return &FCM{client: client, endpoint: endpoint}
}

// Send shows a notification on one device.
func (f *FCM) Send(ctx context.Context, token, title, body string) error {
	if f == nil {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"message": map[string]any{
			"token":        token,
			"notification": map[string]string{"title": title, "body": body},
			"android": map[string]any{
				"notification": map[string]string{
					"channel_id": "superatlas-friends",
					"icon":       "ic_notification",
				},
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	// FCM answers 404 UNREGISTERED (or 400 INVALID_ARGUMENT for a malformed
	// token) when a token will never work again.
	if resp.StatusCode == http.StatusNotFound || strings.Contains(string(msg), "UNREGISTERED") {
		return ErrUnregistered
	}
	return fmt.Errorf("FCM %d: %s", resp.StatusCode, bytes.TrimSpace(msg))
}
