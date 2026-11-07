package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppleAppSiteAssociation(t *testing.T) {
	// Nothing configured: 404, so iOS doesn't cache an empty association.
	rec := httptest.NewRecorder()
	appleAppSiteAssociation(nil)(rec, httptest.NewRequest(http.MethodGet, "/.well-known/apple-app-site-association", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unconfigured: status %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	appleAppSiteAssociation([]string{"TEAM.com.superatlasapp"})(rec, httptest.NewRequest(http.MethodGet, "/.well-known/apple-app-site-association", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q", ct)
	}
	var body struct {
		Applinks struct {
			Details []struct {
				AppIDs     []string            `json:"appIDs"`
				Components []map[string]string `json:"components"`
			} `json:"details"`
		} `json:"applinks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	d := body.Applinks.Details
	if len(d) != 1 || len(d[0].AppIDs) != 1 || d[0].AppIDs[0] != "TEAM.com.superatlasapp" {
		t.Fatalf("details %+v", d)
	}
	if len(d[0].Components) != 1 || d[0].Components[0]["/"] != "/i/*" {
		t.Fatalf("components %+v", d[0].Components)
	}
}

func TestAssetLinks(t *testing.T) {
	rec := httptest.NewRecorder()
	assetLinks(nil)(rec, httptest.NewRequest(http.MethodGet, "/.well-known/assetlinks.json", nil))
	if rec.Body.String() != "[]" {
		t.Fatalf("unconfigured: %q", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	assetLinks([]string{"AB:CD"})(rec, httptest.NewRequest(http.MethodGet, "/.well-known/assetlinks.json", nil))
	var body []struct {
		Target struct {
			PackageName  string   `json:"package_name"`
			Fingerprints []string `json:"sha256_cert_fingerprints"`
		} `json:"target"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body[0].Target.PackageName != androidPackage || body[0].Target.Fingerprints[0] != "AB:CD" {
		t.Fatalf("body %+v", body)
	}
}
