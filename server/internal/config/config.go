package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ListenAddr      string
	DatabasePath    string
	SMTPEmail       string
	SMTPAppPassword string
	// PublicOrigin is the externally reachable base URL (e.g.
	// https://atlas.example.com) used to build invite links. Empty in local
	// dev, where links fall back to the request's own host.
	PublicOrigin string
	// AndroidCertSHA256 lists the release signing cert fingerprints served in
	// /.well-known/assetlinks.json so invite links open the app directly.
	AndroidCertSHA256 []string
	// TrustProxy believes X-Forwarded-For on loopback connections (a local
	// tunnel or reverse proxy) for rate-limit keys.
	TrustProxy bool
	// FCMCredentialsFile is a Firebase service-account key; set, the Google
	// Play build gets friend notifications through FCM.
	FCMCredentialsFile string
	// WebDir holds the built web client (app/dist), served at /. Empty: no
	// web client.
	WebDir string
	// Dev enables debug-only behavior: the "larp" fake-friends trigger, and
	// logging login codes when SMTP isn't configured.
	Dev bool
}

func Load() *Config {
	// .env is optional; real deployments set env vars directly.
	_ = godotenv.Load()

	dataDir := getEnv("SUPERATLAS_DATA_DIR", ".data")
	return &Config{
		ListenAddr:         getEnv("SUPERATLAS_LISTEN_ADDR", ":8080"),
		DatabasePath:       getEnv("SUPERATLAS_DB_PATH", filepath.Join(dataDir, "superatlas.db")),
		SMTPEmail:          os.Getenv("SUPERATLAS_SMTP_EMAIL"),
		SMTPAppPassword:    os.Getenv("SUPERATLAS_SMTP_APP_PASSWORD"),
		PublicOrigin:       strings.TrimRight(os.Getenv("SUPERATLAS_PUBLIC_ORIGIN"), "/"),
		AndroidCertSHA256:  splitList(os.Getenv("SUPERATLAS_ANDROID_CERT_SHA256")),
		TrustProxy:         os.Getenv("SUPERATLAS_TRUST_PROXY") == "1",
		FCMCredentialsFile: os.Getenv("SUPERATLAS_FCM_CREDENTIALS_FILE"),
		WebDir:             os.Getenv("SUPERATLAS_WEB_DIR"),
		Dev:                os.Getenv("SUPERATLAS_DEV") == "1",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
