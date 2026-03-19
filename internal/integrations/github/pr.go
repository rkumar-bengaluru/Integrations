package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/rkumar-bengaluru/Integrations/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/internal/models"
	"go.uber.org/zap"
)

func (h *GitHubHandler) GetPullRequest(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	config *GitHubRuntimeConfig,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("Executing GitHub action", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("Input validation successful")

	// Extract sys_id
	owner := config.Owner
	// Extract repo
	repo := config.Repo

	// Extract installation_id
	installation_id := config.InstallationID
	h.logger.Debug("owner of the repo , and the repo name",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("installation_id", installation_id),
	)

	numberFloat, ok := inputs["number"].(float64) // JSON numbers often come as float64
	if !ok {
		return nil, fmt.Errorf("missing or invalid number in inputs")
	}
	number := int(numberFloat)

	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return nil, err
	}

	client, err := h.buildGitHubClient(config)
	if err != nil {
		return nil, err
	}

	// Call GitHub API
	pr, resp, err := client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to get pull request: %w", err),
		}, nil
	}

	// Map PR fields to output schema
	result := map[string]interface{}{
		"id":        pr.GetID(),
		"number":    pr.GetNumber(),
		"title":     pr.GetTitle(),
		"body":      pr.GetBody(),
		"state":     pr.GetState(),
		"url":       pr.GetURL(),
		"patch_url": pr.GetPatchURL(),
		"user": map[string]interface{}{
			"login":      pr.GetUser().GetLogin(),
			"id":         pr.GetUser().GetID(),
			"avatar_url": pr.GetUser().GetAvatarURL(),
		},
	}

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, result, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	} else {
		h.logger.Debug("Output validation successful")
	}

	diffs, err := h.GetPullRequestDiffs(ctx, config, client, owner, repo, number)
	if err != nil {
		return nil, err
	}

	for _, d := range diffs {
		fmt.Printf("File: %s\n", d.FileName)
		for _, h := range d.Hunks {
			fmt.Println(h.Header)
			fmt.Println("Added lines:")
			for _, l := range h.Added {
				fmt.Println(l)
			}
			fmt.Println("Removed lines:")
			for _, l := range h.Removed {
				fmt.Println(l)
			}
		}
	}

	// Return ActionResult
	return &handler.ActionResult{
		Data:       result,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (h *GitHubHandler) ListPullRequests(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	config *GitHubRuntimeConfig,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {
	h.logger.Debug("finally action time for ServiceNow", zap.String("action", string(actionDef.Type)))

	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return nil, err
	}

	client, err := h.buildGitHubClient(config)
	if err != nil {
		return nil, err
	}

	commons.PrintCollectedParams(inputs)
	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	// Extract sys_id
	owner := config.Owner
	// Extract repo
	repo := config.Repo

	// Extract installation_id
	installation_id := config.InstallationID
	h.logger.Debug("owner of the repo , and the repo name",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("installation_id", installation_id),
	)

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("owner and repo are required (provided: owner=%s, repo=%s)", owner, repo)
	}

	// Extract installation_id
	state, ok := inputs["state"].(string)
	if !ok || state == "" {
		return nil, fmt.Errorf("missing or invalid state in inputs")
	}
	// Extract per_page
	perPageFloat, ok := inputs["per_page"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid per_page in inputs")
	}
	per_page := int(perPageFloat)

	// Extract per_page
	pageFloat, ok := inputs["page"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid page in inputs")
	}

	pageNo := int(pageFloat)

	opts := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: per_page,
			Page:    pageNo,
		},
	}

	// Optional filters
	if head, ok := inputs["head"].(string); ok && head != "" {
		opts.Head = head
	}
	if base, ok := inputs["base"].(string); ok && base != "" {
		opts.Base = base
	}
	if sort, ok := inputs["sort"].(string); ok && sort != "" {
		opts.Sort = sort
	}
	if direction, ok := inputs["direction"].(string); ok && direction != "" {
		opts.Direction = direction
	}

	pullRequests, resp, err := client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %w", err)
	}

	prsData := make([]map[string]interface{}, len(pullRequests))
	for i, pr := range pullRequests {
		prsData[i] = pullRequestToMap(pr)
	}

	h.logger.Debug("output validation successful")

	result := &handler.ActionResult{
		Data:       prsData,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"rate_limit":     resp.Rate.Limit,
			"rate_remaining": resp.Rate.Remaining,
		},
	}
	// Validate outputs
	if err := commons.ValidateSchemaSupportsArray(actionDef.OutputSchema, result.Data, "output"); err != nil {
		// Log warning but don't fail - schema might be stricter than actual data
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
		// Or return error if strict validation required:
		// return nil, fmt.Errorf("output validation failed: %w", err)
	}

	return result, nil
}

func pullRequestToMap(pr *github.PullRequest) map[string]interface{} {
	if pr == nil {
		return nil
	}

	result := map[string]interface{}{
		"url":    pr.GetURL(),
		"id":     pr.GetID(),     // This is int64, need to handle
		"number": pr.GetNumber(), // This is int
		"state":  pr.GetState(),
		"title":  pr.GetTitle(),
		"body":   pr.GetBody(),
		// "html_url":  pr.HTMLURL, // web link
		// "diff_url":  pr.DiffURL,
		"patch_url": pr.PatchURL,
		// "issue_url": pr.IssueURL,

	}

	// Handle int64 -> int conversion for JSON compatibility
	if id := pr.GetID(); id != 0 {
		result["id"] = int(id) // Convert int64 to int
	}

	if num := pr.GetNumber(); num != 0 {
		result["number"] = num
	}

	// Handle nested user object
	if pr.User != nil {
		result["user"] = userToMap(pr.User)
	}

	return result
}

func formatTimestamp(t *github.Timestamp) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func userToMap(user *github.User) map[string]interface{} {
	if user == nil {
		return nil
	}
	return map[string]interface{}{
		"login":      user.GetLogin(),
		"id":         user.GetID(),
		"node_id":    user.GetNodeID(),
		"avatar_url": user.GetAvatarURL(),
		"type":       user.GetType(),
		"site_admin": user.GetSiteAdmin(),
	}
}
