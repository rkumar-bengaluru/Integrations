package actions

import (
	"context"

	"agent.fabric.com/modules/internal/models"
	"agent.fabric.com/modules/internal/repository"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func CreateIncidentAction(ctx context.Context,
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
		Description:       "Create a new incident in ServiceNow",
		Type:              models.ActionType(actionType),
		SchemaVersion:     "v1",
		SupportsStreaming: false,
		IsInternal:        false,
		Version:           "1.0",
		IsActive:          true,
		ActionHandler:     "servicenow_create_incident",
		InputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Brief summary of the incident",
					"source":      "user_input",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Detailed description of the incident",
					"source":      "user_input",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Incident category (e.g., software, hardware)",
					"source":      "user_input",
				},
				"subcategory": map[string]interface{}{
					"type":        "string",
					"description": "Incident subcategory (e.g., CRM, network)",
					"source":      "user_input",
				},
				"caller_id": map[string]interface{}{
					"type":        "string",
					"description": "User ID or email of the caller",
					"source":      "user_input",
				},
				"impact": map[string]interface{}{
					"type":        "string",
					"description": "Impact level (1=High, 2=Medium, 3=Low)",
					"source":      "user_input",
				},
				"urgency": map[string]interface{}{
					"type":        "string",
					"description": "Urgency level (1=High, 2=Medium, 3=Low)",
					"source":      "user_input",
				},
				"assignment_group": map[string]interface{}{
					"type":        "string",
					"description": "Group to assign the incident to",
					"source":      "user_input",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Incident state (e.g., New, In Progress)",
					"source":      "user_input",
				},
				"contact_type": map[string]interface{}{
					"type":        "string",
					"description": "How the incident was reported (phone, chat, email)",
					"source":      "user_input",
				},
				"u_channel": map[string]interface{}{
					"type":        "string",
					"description": "Channel through which issue was reported (Slack, Teams)",
					"source":      "user_input",
				},
			},
			"required": []string{"short_description", "description", "category", "caller_id"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "string",
					"description": "Incident number assigned by ServiceNow",
				},
				"sys_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique system identifier for the incident",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Current state of the incident",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority calculated from impact and urgency",
				},
				"assignment_group": map[string]interface{}{
					"type":        "string",
					"description": "Group assigned to the incident",
				},
				"caller_id": map[string]interface{}{
					"type":        "string",
					"description": "Caller associated with the incident",
				},
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Incident summary",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Incident details",
				},
			},
			"required": []interface{}{"number", "sys_id", "state"},
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

func GetIncidentByNumberAction(ctx context.Context,
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
		Description:       "Retrieve an incident from ServiceNow by its number",
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
					"type":        "string",
					"description": "Incident number (e.g., INC0010107)",
					"source":      "user_input",
				},
			},
			"required": []string{"number"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "string",
					"description": "Incident number assigned by ServiceNow",
				},
				"sys_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique system identifier for the incident",
				},
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Brief summary of the incident",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Current state of the incident",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority calculated from impact and urgency",
				},
			},
			"required": []interface{}{"number", "sys_id"},
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

func SearchSimilarIncidentAction(ctx context.Context,
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
		Description:       "Search for incidents in ServiceNow using keywords and filters",
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
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Encoded sysparm_query string for incident search (e.g., short_descriptionLIKEsalesforce^ORdescriptionLIKEsalesforce^opened_atONLast7days@javascript:gs.beginningOfLast7Days())",
					"source":      "user_input",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of incidents to return",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "string",
					"description": "Incident number assigned by ServiceNow",
				},
				"sys_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique system identifier for the incident",
				},
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Brief summary of the incident",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Current state of the incident",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Priority calculated from impact and urgency",
				},
			},
			"required": []interface{}{"number", "sys_id"},
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

func UpdateIncidentAction(ctx context.Context,
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
		Description:       "Update an existing incident in ServiceNow using its sys_id",
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
				"sys_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique system identifier of the incident to update",
					"source":      "user_input",
				},
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Updated brief summary of the incident",
					"source":      "user_input",
				},
			},
			"required": []string{"sys_id", "short_description"},
		},
		OutputSchema: models.JSONMap{
			"type": "object",
			"properties": map[string]interface{}{
				"number": map[string]interface{}{
					"type":        "string",
					"description": "Incident number assigned by ServiceNow",
				},
				"sys_id": map[string]interface{}{
					"type":        "string",
					"description": "Unique system identifier for the incident",
				},
				"short_description": map[string]interface{}{
					"type":        "string",
					"description": "Updated incident summary",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "Current state of the incident",
				},
				"priority": map[string]interface{}{
					"type":        "string",
					"description": "Updated priority of the incident",
				},
				"assignment_group": map[string]interface{}{
					"type":        "string",
					"description": "Group assigned to the incident",
				},
			},
			"required": []interface{}{"number", "sys_id"},
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
