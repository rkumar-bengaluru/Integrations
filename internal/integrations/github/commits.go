package github

import (
	"context"
	"fmt"

	"github.com/rkumar-bengaluru/Integrations/v2/internal/handler"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/integrations/commons"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"go.uber.org/zap"
)

func (h *GitHubHandler) ListCommits(
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

	numberFloat, ok := inputs["number"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid number in inputs")
	}
	number := int(numberFloat)

	// Resolve runtime config and client
	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return nil, err
	}
	client, err := h.buildGitHubClient(config)
	if err != nil {
		return nil, err
	}

	// Call GitHub API
	commits, resp, err := client.PullRequests.ListCommits(ctx, owner, repo, number, nil)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to list commits: %w", err),
		}, nil
	}

	// Map commits to output schema
	var result []map[string]interface{}
	for _, c := range commits {
		result = append(result, map[string]interface{}{
			"sha":     c.GetSHA(),
			"message": c.GetCommit().GetMessage(),
			"url":     c.GetURL(),
			"author": map[string]interface{}{
				"login":      c.GetAuthor().GetLogin(),
				"id":         c.GetAuthor().GetID(),
				"avatar_url": c.GetAuthor().GetAvatarURL(),
			},
		})
	}

	// Validate outputs
	if err := commons.ValidateSchemaSupportsArray(actionDef.OutputSchema, result, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	} else {
		h.logger.Debug("Output validation successful")
	}

	return &handler.ActionResult{
		Data:       result,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (h *GitHubHandler) GetCommit(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	config *GitHubRuntimeConfig,
	binding models.IntegrationBinding,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("Executing GitHub action", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate inputs
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

	sha, ok := inputs["sha"].(string)
	if !ok || sha == "" {
		return nil, fmt.Errorf("missing or invalid sha in inputs")
	}

	// Resolve runtime config and client
	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return nil, err
	}
	client, err := h.buildGitHubClient(config)
	if err != nil {
		return nil, err
	}

	// Call GitHub API
	commit, resp, err := client.Repositories.GetCommit(ctx, owner, repo, sha, nil)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to get commit: %w", err),
		}, nil
	}

	// Map commit fields
	var files []map[string]interface{}
	for _, f := range commit.Files {
		files = append(files, map[string]interface{}{
			"filename": f.GetFilename(),
			"status":   f.GetStatus(),
			"patch":    f.GetPatch(),
		})
	}

	result := map[string]interface{}{
		"sha":     commit.GetSHA(),
		"message": commit.Commit.GetMessage(),
		"url":     commit.GetURL(),
		"files":   files,
	}

	// Validate outputs
	if err := commons.ValidateSchema(actionDef.OutputSchema, result, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	} else {
		h.logger.Debug("Output validation successful")
	}

	return &handler.ActionResult{
		Data:       result,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}
