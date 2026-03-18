package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/v84/github"
	"go.uber.org/zap"
)

func (h *GitHubHandler) GetPullRequestDiffs(
	ctx context.Context,
	config *GitHubRuntimeConfig,
	client *github.Client,
	owner, repo string,
	number int,
) ([]FileDiff, error) {

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d.patch", owner, repo, number)
	h.logger.Debug("url", zap.String("", url))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Use same transport as your GitHub client

	// Critical: These headers must be set exactly
	// Critical: Set these headers to get raw diff text
	req.Header.Set("Authorization", "Bearer "+config.Token)
	req.Header.Set("Accept", "application/vnd.github.v3.patch") // This triggers the text output
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Debug logging
	h.logger.Debug("diff request",
		zap.String("url", url),
		zap.String("token", config.Token),
		zap.Int("status", resp.StatusCode),
		zap.String("content-type", resp.Header.Get("Content-Type")),
	)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, string(body))
	}

	rawDiff, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	h.logger.Debug("raw diff received",
		zap.Int("bytes", len(rawDiff)),
		zap.String("preview", string(rawDiff[:min(100, len(rawDiff))])),
	)

	return h.parseDiff(string(rawDiff)), nil
}

func (h *GitHubHandler) GetPullRequestDiffs1(
	ctx context.Context,
	config *GitHubRuntimeConfig,
	client *github.Client,
	owner, repo string,
	number int,
) ([]FileDiff, error) {

	// DEBUG: Check what's happening
	h.logger.Debug("client state",
		zap.String("base_url", func() string {
			if client.BaseURL == nil {
				return "nil (default: https://api.github.com/)"
			}
			return client.BaseURL.String()
		}()),
		zap.String("token_preview", config.Token[:min(10, len(config.Token))]+"..."),
	)

	// Use the dedicated raw method with "diff" or "patch" media type
	rawDiff, resp, err := client.PullRequests.GetRaw(ctx, owner, repo, number, github.RawOptions{
		Type: github.Diff, // or github.Patch
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get PR diff: %w", err)
	}
	defer resp.Body.Close()
	h.logger.Debug("raw diff", zap.Any("", rawDiff))
	// rawDiff is a string containing the full diff
	return h.parseDiff(rawDiff), nil
}

// parseDiff parses unified diff format into structured data
func (h *GitHubHandler) parseDiff(raw string) []FileDiff {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	chunks := strings.Split(raw, "diff --git ")
	var diffs []FileDiff

	for _, chunk := range chunks[1:] { // Skip first empty chunk
		diff := h.parseDiffChunk("diff --git " + chunk)
		diffs = append(diffs, diff)
	}

	return diffs
}

func (h *GitHubHandler) parseDiffChunk(raw string) FileDiff {
	lines := strings.Split(raw, "\n")

	var fileName string
	var hunks []Hunk
	var current *Hunk

	for _, line := range lines {
		// Extract filename from +++ line
		if strings.HasPrefix(line, "+++ b/") {
			fileName = strings.TrimPrefix(line, "+++ b/")
			continue
		}

		// New hunk
		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}
			current = &Hunk{Header: line}
			continue
		}

		if current == nil {
			continue
		}

		// Classify line
		switch {
		case strings.HasPrefix(line, "+"):
			current.Added = append(current.Added, line[1:]) // Strip + prefix
		case strings.HasPrefix(line, "-"):
			current.Removed = append(current.Removed, line[1:]) // Strip - prefix
		default:
			current.Context = append(current.Context, line)
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return FileDiff{
		FileName: fileName,
		Hunks:    hunks,
	}
}
