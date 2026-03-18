package github

import (
	"net/http"
	"time"

	"agent.fabric.com/modules/internal/models"
)

// RuntimeConfig holds resolved values from binding
type GitHubRuntimeConfig struct {
	Token          string
	TokenExpiresAt time.Time
	PrivateKey     string
	AppID          string
	InstallationID string
	Owner          string
	Repo           string
	BaseURL        string
	APIVersion     string
	Credential     *models.Credential
}

type githubTokenTransport struct {
	token         string
	apiVersion    string
	baseTransport http.RoundTripper
}

type Hunk struct {
	Header  string   // e.g. "@@ -10,7 +10,9 @@ func Example() {"
	Added   []string // lines starting with '+'
	Removed []string // lines starting with '-'
	Context []string // unchanged context lines
}

type FileDiff struct {
	FileName string
	Hunks    []Hunk
}
