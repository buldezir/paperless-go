package e2e

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAuthUserLogin(t *testing.T) {
	h := StartShared(t)
	auth := h.authWithPassword(t, "users", UserEmail, UserPassword)
	if auth.Token == "" {
		t.Fatal("expected token")
	}
	if jsonGetString(auth.Record, "email") != UserEmail {
		t.Fatalf("email=%q", auth.Record["email"])
	}
}

func TestAuthSuperuserLogin(t *testing.T) {
	h := StartShared(t)
	auth := h.authWithPassword(t, "_superusers", SuperEmail, SuperPassword)
	if auth.Token == "" {
		t.Fatal("expected token")
	}
}

func TestAuthBadPassword(t *testing.T) {
	h := StartShared(t)
	status, raw := h.doJSON(t, http.MethodPost, "/api/collections/users/auth-with-password", "", map[string]string{
		"identity": UserEmail,
		"password": "wrong-password",
	})
	if status == http.StatusOK {
		t.Fatalf("expected auth failure, got %s", raw)
	}
}

func TestUnauthenticatedDocumentsRejected(t *testing.T) {
	h := StartShared(t)
	status, raw := h.doJSON(t, http.MethodGet, "/api/collections/documents/records", "", nil)
	if status == http.StatusOK {
		t.Fatalf("expected rejection, got %s", raw)
	}
	if status != http.StatusUnauthorized && status != http.StatusForbidden {
		// PocketBase may return 400 with auth error depending on rules/guard.
		var body map[string]any
		_ = json.Unmarshal([]byte(raw), &body)
		if status < 400 {
			t.Fatalf("expected client error, got %d %s", status, raw)
		}
	}
}
