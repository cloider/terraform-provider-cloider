package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	DefaultAPIURL  = "https://cloider.com/api"
	defaultProfile = "default"
)

// Settings is what the provider block sets, empty when it isn't.
type Settings struct {
	APIURL    string
	Token     string
	TokenFile string
	Profile   string
}

// Session is the api url and token the provider talks with.
type Session struct {
	APIURL string
	Token  string
	// Where the token came from, for error messages.
	Source string
}

// The CLI's ~/.cloider/config.json, only what finding a profile's token needs.
// It mirrors cli/internal/config/config.go.
type cliConfig struct {
	CurrentProfile string `json:"current_profile"`
	Profiles       map[string]struct {
		APIURL    string `json:"api_url"`
		TokenFile string `json:"token_file"`
	} `json:"profiles"`
}

var profileName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// Resolve finds the api url and token. The first token source that is set wins:
// token, token_file, CLOIDER_TOKEN, CLOIDER_TOKEN_FILE, then the token file
// `cloider login` wrote for the CLI profile in use.
func Resolve(s Settings) (*Session, error) {
	dir, err := cliConfigDir()
	if err != nil {
		return nil, err
	}
	cfg, err := loadCLIConfig(dir)
	if err != nil {
		return nil, err
	}

	profile := firstNonEmpty(s.Profile, os.Getenv("CLOIDER_PROFILE"), cfg.CurrentProfile, defaultProfile)
	if !profileName.MatchString(profile) {
		return nil, fmt.Errorf("invalid profile name %q: use letters, digits, '.', '_' or '-'", profile)
	}
	p := cfg.Profiles[profile]

	session := &Session{APIURL: strings.TrimRight(firstNonEmpty(s.APIURL, os.Getenv("CLOIDER_API_URL"), p.APIURL, DefaultAPIURL), "/")}

	switch {
	case s.Token != "":
		session.Token, session.Source = strings.TrimSpace(s.Token), "the provider's token"
	case s.TokenFile != "":
		session.Source = "token_file " + s.TokenFile
		session.Token, err = readTokenFile(expandHome(s.TokenFile))
	case os.Getenv("CLOIDER_TOKEN") != "":
		session.Token, session.Source = strings.TrimSpace(os.Getenv("CLOIDER_TOKEN")), "CLOIDER_TOKEN"
	case os.Getenv("CLOIDER_TOKEN_FILE") != "":
		path := os.Getenv("CLOIDER_TOKEN_FILE")
		session.Source = "CLOIDER_TOKEN_FILE " + path
		session.Token, err = readTokenFile(expandHome(path))
	default:
		path := filepath.Join(dir, "token")
		if profile != defaultProfile {
			path = filepath.Join(dir, "tokens", profile)
		}
		if p.TokenFile != "" {
			path = expandHome(p.TokenFile)
		}
		session.Source = fmt.Sprintf("the CLI profile %s (%s)", profile, path)
		session.Token, err = readTokenFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, errors.New("no API token: set token or CLOIDER_TOKEN, or run `cloider login`")
		}
	}
	if err != nil {
		return nil, fmt.Errorf("can't read the token from %s: %w", session.Source, err)
	}
	return session, nil
}

func cliConfigDir() (string, error) {
	if dir := os.Getenv("CLOIDER_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("can't find the home directory: %w", err)
	}
	return filepath.Join(home, ".cloider"), nil
}

// loadCLIConfig reads config.json. A missing one is an empty config, the CLI
// may never have been used here.
func loadCLIConfig(dir string) (*cliConfig, error) {
	cfg := &cliConfig{}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s is not valid json: %w", filepath.Join(dir, "config.json"), err)
	}
	return cfg, nil
}

func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return token, nil
}

func expandHome(path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, rest)
		}
	}
	return path
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
