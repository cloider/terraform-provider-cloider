package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setup points the CLI's config dir at a temp dir and clears the env the
// resolution reads, so the developer's own login never leaks into a test.
func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLOIDER_CONFIG_DIR", dir)
	for _, key := range []string{"CLOIDER_TOKEN", "CLOIDER_TOKEN_FILE", "CLOIDER_PROFILE", "CLOIDER_API_URL"} {
		t.Setenv(key, "")
	}
	return dir
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveOrder(t *testing.T) {
	dir := setup(t)
	write(t, filepath.Join(dir, "token"), "from-profile\n")
	attrFile := filepath.Join(t.TempDir(), "attr-token")
	write(t, attrFile, "from-attr-file")
	envFile := filepath.Join(t.TempDir(), "env-token")
	write(t, envFile, "from-env-file")

	check := func(s Settings, want string) {
		t.Helper()
		session, err := Resolve(s)
		if err != nil {
			t.Fatal(err)
		}
		if session.Token != want {
			t.Fatalf("token %q, want %q (from %s)", session.Token, want, session.Source)
		}
	}

	check(Settings{}, "from-profile")
	t.Setenv("CLOIDER_TOKEN_FILE", envFile)
	check(Settings{}, "from-env-file")
	t.Setenv("CLOIDER_TOKEN", "from-env")
	check(Settings{}, "from-env")
	check(Settings{TokenFile: attrFile}, "from-attr-file")
	check(Settings{Token: "from-attr", TokenFile: attrFile}, "from-attr")
}

func TestResolveProfile(t *testing.T) {
	dir := setup(t)
	write(t, filepath.Join(dir, "tokens", "acme"), "acme-token")
	write(t, filepath.Join(dir, "config.json"), `{"current_profile":"acme","profiles":{"acme":{"api_url":"https://localhost/api/"}}}`)

	session, err := Resolve(Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if session.Token != "acme-token" || session.APIURL != "https://localhost/api" {
		t.Fatalf("got %+v", session)
	}

	session, err = Resolve(Settings{APIURL: "http://127.0.0.1:8000/api"})
	if err != nil || session.APIURL != "http://127.0.0.1:8000/api" {
		t.Fatalf("api_url should win over the profile's: %+v, %v", session, err)
	}
}

func TestResolveWithoutToken(t *testing.T) {
	setup(t)
	_, err := Resolve(Settings{})
	if err == nil || !strings.Contains(err.Error(), "cloider login") {
		t.Fatalf("expected a hint to log in, got %v", err)
	}

	if _, err := Resolve(Settings{TokenFile: filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatal("a token_file that isn't there is an error")
	}
}
