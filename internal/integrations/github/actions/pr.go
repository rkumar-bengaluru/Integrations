package actions

import (
	"context"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/google/uuid"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/models"
	"github.com/rkumar-bengaluru/Integrations/v2/internal/repository"
	"go.uber.org/zap"
)

func ListPullRequestAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "List pull requests for a repository",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"owner": map[string]interface{}{
					"type":        "string",
					"description": "Repository owner",
					"source":      "$.bindings.credential.owner",
				},
				"repo": map[string]interface{}{
					"type":        "string",
					"description": "Repository name",
					"source":      "$.bindings.credential.repo",
				},
				"state": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"open", "closed", "all"},
					"default": "open",
				},
				"sort": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"created", "updated", "popularity", "long-running"},
					"default": "created",
				},
				"direction": map[string]interface{}{
					"type":    "string",
					"enum":    []string{"asc", "desc"},
					"default": "desc",
				},
				"per_page": map[string]interface{}{
					"type":    "integer",
					"default": 30,
					"maximum": 100,
				},
				"page": map[string]interface{}{
					"type":    "integer",
					"default": 1,
				},
			},
			"required": []string{},
		},
		OutputSchema: models.JSONMap{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":        map[string]interface{}{"type": "string"},
					"id":         map[string]interface{}{"type": "integer"},
					"number":     map[string]interface{}{"type": "integer"},
					"state":      map[string]interface{}{"type": "string"},
					"title":      map[string]interface{}{"type": "string"},
					"body":       map[string]interface{}{"type": "string"},
					"created_at": map[string]interface{}{"type": "string"},
					"updated_at": map[string]interface{}{"type": "string"},
					"closed_at":  map[string]interface{}{"type": "string"},
					"merged_at":  map[string]interface{}{"type": "string"},
					"user": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"login": map[string]interface{}{"type": "string"},
							"id":    map[string]interface{}{"type": "integer"},
						},
					},
				},
				"required": []interface{}{"url", "id", "number", "state", "title", "created_at", "updated_at", "user"},
				// optional but recommended:
				// "additionalProperties": false,
			},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func GetPullRequestByNumberAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Retrieve details of a single GitHub pull request by number",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "number",
					"description": "Pull request number (e.g., 2)",
					"source":      "user_input",
				},
			},
			"required": []string{"number"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "integer",
					"description": "Unique ID of the pull request",
				},
				"number": map[string]interface{}{
					"type":        "integer",
					"description": "Pull request number",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title of the pull request",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Description text of the pull request",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "State of the pull request (open, closed, merged)",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "API URL of the pull request",
				},
				"patch_url": map[string]interface{}{
					"type":        "string",
					"description": "Direct URL to the patch file",
				},
				"user": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"login": map[string]interface{}{
							"type":        "string",
							"description": "GitHub username of the PR author",
						},
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "User ID of the PR author",
						},
						"avatar_url": map[string]interface{}{
							"type":        "string",
							"description": "Avatar URL of the PR author",
						},
					},
				},
			},
			"required": []interface{}{"id", "number", "title", "state", "url"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func ListCommitsByPRAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "List all commits associated with a pull request",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "number",
					"description": "Pull request number (e.g., 2)",
					"source":      "user_input",
				},
			},
			"required": []string{"number"},
		},
		OutputSchema: models.JSONMap{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sha": map[string]interface{}{
						"type":        "string",
						"description": "Commit SHA",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Commit message",
					},
					"url": map[string]interface{}{
						"type":        "string",
						"description": "API URL of the commit",
					},
					"author": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"login": map[string]interface{}{
								"type":        "string",
								"description": "GitHub username of the commit author",
							},
							"id": map[string]interface{}{
								"type":        "integer",
								"description": "User ID of the commit author",
							},
							"avatar_url": map[string]interface{}{
								"type":        "string",
								"description": "Avatar URL of the commit author",
							},
						},
					},
				},
			},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func GetCommitsByShaAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Retrieve details of a specific commit in a repository",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"sha": map[string]interface{}{
					"type":        "string",
					"description": "Commit SHA (e.g., abc123...)",
					"source":      "user_input",
				},
			},
			"required": []string{"sha"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"sha": map[string]interface{}{
					"type":        "string",
					"description": "Commit SHA",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Commit message",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "API URL of the commit",
				},
				"files": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"filename": map[string]interface{}{
								"type":        "string",
								"description": "Name of the file changed in this commit",
							},
							"status": map[string]interface{}{
								"type":        "string",
								"description": "Change status (added, modified, removed)",
							},
							"patch": map[string]interface{}{
								"type":        "string",
								"description": "Diff patch for the file",
							},
						},
					},
				},
			},
			"required": []interface{}{"sha", "message", "url"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}

func CreateReviewCommentAction(ctx context.Context,
	tenantID uuid.UUID, actionName, actionType string,
	integrationRepo repository.IntegrationRepository,
	integration *models.Integration) (*models.ActionDefinition, error) {

	resp, _ := integrationRepo.CheckIfActionDefinitionExist(ctx, integrationRepo, actionType, actionName, tenantID)

	if resp != nil {
		logger.Debug("action definition already present returning", zap.String(actionType, actionName))
		return resp, nil
	}

	action := &models.ActionDefinition{
		TenantID:          tenantID,
		Name:              actionName,
		Description:       "Add a review comment to a pull request, either general or inline on a specific file/line",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     actionType,
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "number",
					"description": "Pull request number (e.g., 2)",
					"source":      "user_input",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Text content of the review comment",
					"source":      "user_input",
				},
				"commit_id": map[string]interface{}{
					"type":        "string",
					"description": "SHA of the commit the comment applies to (required for inline comments)",
					"source":      "user_input",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path in the repository (required for inline comments)",
					"source":      "user_input",
				},
				"position": map[string]interface{}{
					"type":        "number",
					"description": "Line index in the diff where the comment should appear (required for inline comments)",
					"source":      "user_input",
				},
			},
			"required": []string{"number", "body"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "integer",
					"description": "Unique ID of the created review comment",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Text content of the review comment",
				},
				"commit_id": map[string]interface{}{
					"type":        "string",
					"description": "Commit SHA the comment is attached to",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path the comment is attached to",
				},
				"position": map[string]interface{}{
					"type":        "integer",
					"description": "Line index in the diff where the comment appears",
				},
				"url": map[string]interface{}{
					"type":        "string",
					"description": "API URL of the created review comment",
				},
				"user": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"login": map[string]interface{}{
							"type":        "string",
							"description": "GitHub username of the commenter",
						},
						"id": map[string]interface{}{
							"type":        "integer",
							"description": "User ID of the commenter",
						},
						"avatar_url": map[string]interface{}{
							"type":        "string",
							"description": "Avatar URL of the commenter",
						},
					},
				},
			},
			"required": []interface{}{"id", "body", "url"},
		},
	}

	logger.Debug("adding action definition", zap.String("", action.Name))

	err := integrationRepo.CreateActionDefinition(ctx, integration.ID, action)
	if err != nil {
		return nil, err
	}

	err = integrationRepo.AddActionToIntegration(ctx, integration.ID, action.ID)

	if err != nil {
		return nil, err
	}

	return action, nil
}
