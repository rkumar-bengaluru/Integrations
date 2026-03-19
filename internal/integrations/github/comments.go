package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/utils"
	"go.uber.org/zap"
)

func (h *GitHubHandler) AddReviewComment(
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

	owner := config.Owner
	repo := config.Repo
	installationID := config.InstallationID
	h.logger.Debug("Repo context",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("installation_id", installationID),
	)

	// Extract required inputs
	numberFloat, ok := inputs["number"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid number in inputs")
	}
	number := int(numberFloat)

	body, ok := inputs["body"].(string)
	if !ok || body == "" {
		return nil, fmt.Errorf("missing or invalid body in inputs")
	}

	// Optional inputs for inline comments
	commitID, _ := inputs["commit_id"].(string)
	path, _ := inputs["path"].(string)
	positionFloat, _ := inputs["position"].(float64)
	position := int(positionFloat)

	// Resolve runtime config and client
	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return nil, err
	}
	client, err := h.buildGitHubClient(config)
	if err != nil {
		return nil, err
	}

	// Build review comment object
	reviewComment := &github.PullRequestComment{
		Body: github.String(body),
	}
	if commitID != "" {
		reviewComment.CommitID = github.String(commitID)
	}
	if path != "" {
		reviewComment.Path = github.String(path)
	}
	if position > 0 {
		reviewComment.Position = github.Int(position)
	}
	reviewComment.Side = utils.Ptr("RIGHT")

	// Call GitHub API
	comment, resp, err := client.PullRequests.CreateComment(ctx, owner, repo, number, reviewComment)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to add review comment: %w", err),
		}, nil
	}

	// Map fields to output schema
	result := map[string]interface{}{
		"id":        comment.GetID(),
		"body":      comment.GetBody(),
		"commit_id": comment.GetCommitID(),
		"path":      comment.GetPath(),
		"position":  comment.GetPosition(),
		"url":       comment.GetURL(),
		"user": map[string]interface{}{
			"login":      comment.GetUser().GetLogin(),
			"id":         comment.GetUser().GetID(),
			"avatar_url": comment.GetUser().GetAvatarURL(),
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
