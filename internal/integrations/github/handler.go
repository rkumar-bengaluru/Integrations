package github

import (
	"context"
	"fmt"

	"agent.fabric.com/modules/internal/encryption"
	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/service"
	"go.uber.org/zap"
)

// GitHubHandler implements all GitHub actions using Integration config
type GitHubHandler struct {
	encryptionSvc encryption.EncryptionService
	bindingSvc    service.IntegrationBindingService
	logger        *zap.Logger
}

// Ensure GitHubHandler implements IntegrationHandler
var _ handler.IntegrationHandler = (*GitHubHandler)(nil)

func NewGitHubHandler(encryptionSvc encryption.EncryptionService, bindingSvc service.IntegrationBindingService, logger *zap.Logger) *GitHubHandler {
	return &GitHubHandler{
		encryptionSvc: encryptionSvc,
		bindingSvc:    bindingSvc,
		logger:        logger,
	}
}

// Init generates the initial installation token for GitHub App authentication
// It takes the collected parameters (private_key, app_id, installation_id, owner, repo)
// and returns the complete credential data including the generated token
func (h *GitHubHandler) Init(
	ctx context.Context,
	config *models.ExecutionConfig,
	collectedParams map[string]interface{}) (map[string]interface{}, error) {
	return h.oauthServer2Server(ctx, config, collectedParams)
}

// Execute an action with runtime-resolved config
func (h *GitHubHandler) Execute(
	ctx context.Context,
	config *models.ExecutionConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding, inputs map[string]interface{}) (*handler.ActionResult, error) {

	runtimeConfig, err := h.resolveRuntimeConfig(ctx, config, binding)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential: %w", err)
	}

	switch string(actionDef.Type) {
	case GithubTestActionType:
		return h.TestAction(ctx, runtimeConfig, actionDef, binding, inputs)
	case GithubPullOpenRequestActionType:
		return h.ListPullRequests(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GithubGetPullRequestByNumberActionType:
		return h.GetPullRequest(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GithubListCommitsActionType:
		return h.ListCommits(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GithubGetCommitsActionType:
		return h.GetCommit(ctx, config, actionDef, runtimeConfig, binding, inputs)
	case GithubAddReviewCommentActionType:
		return h.AddReviewComment(ctx, config, actionDef, runtimeConfig, binding, inputs)
	default:
		return nil, fmt.Errorf("unsupported action: %s", actionDef.Type)
	}
}

// Test if binding configuration is valid
func (h *GitHubHandler) TestConnection(
	ctx context.Context,
	econfig *models.ExecutionConfig,
	binding models.IntegrationBinding,
) error {
	config, err := h.resolveRuntimeConfig(ctx, econfig, binding)
	if err != nil {
		return err
	}

	client, err := h.buildGitHubClient(config)
	if err != nil {
		return err
	}

	// Try to list repos for the installation
	repos, resp, err := client.Apps.ListRepos(ctx, nil)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	h.logger.Debug("GitHub connection test successful",
		zap.Int("repo_count", *repos.TotalCount),
		zap.Int("rate_remaining", resp.Rate.Remaining),
	)

	return nil
}
