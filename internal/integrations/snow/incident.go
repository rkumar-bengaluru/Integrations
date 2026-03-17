package snow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"agent.fabric.com/modules/internal/handler"
	"agent.fabric.com/modules/internal/integrations/commons"
	"agent.fabric.com/modules/internal/models"
	"go.uber.org/zap"
)

func (h *SnowHandler) CreateIncident(
	ctx context.Context,
	runtimeConfig *SnowRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	config *models.ExecutionConfig,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("finally action time for ServiceNow", zap.String("action", string(actionDef.Type)))

	commons.PrintCollectedParams(inputs)

	// Validate required fields
	if runtimeConfig.AccessToken == nil || *runtimeConfig.AccessToken == "" {
		return nil, fmt.Errorf("missing AccessToken for ServiceNow flow")
	}

	instanceURL := strings.Split(*config.CredentialBinding.AuthorityUrl, "/oauth_token.do")[0]

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	// Prepare request payload (map directly from inputs)
	payload := map[string]interface{}{
		"short_description": inputs["short_description"],
		"description":       inputs["description"],
		"category":          inputs["category"],
		"subcategory":       inputs["subcategory"],
		"caller_id":         inputs["caller_id"],
		"impact":            inputs["impact"],
		"urgency":           inputs["urgency"],
		"assignment_group":  inputs["assignment_group"],
		"state":             inputs["state"],
		"contact_type":      inputs["contact_type"],
		"u_channel":         inputs["u_channel"],
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to marshal payload: %w", err),
		}, nil
	}

	// Build request
	url := fmt.Sprintf("%s/api/now/table/incident", instanceURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call ServiceNow API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	// Decode ServiceNow response
	var snowResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&snowResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode ServiceNow response: %w", err),
		}, nil
	}

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, snowResp, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	}

	h.logger.Debug("Output validation successful")

	// Check for ServiceNow error (e.g., missing result)
	if _, exists := snowResp["result"]; !exists {
		return &handler.ActionResult{
			Data:       snowResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: missing result object"),
		}, nil
	}

	// Return ActionResult with full ServiceNow response
	return &handler.ActionResult{
		Data:       snowResp,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (h *SnowHandler) GetIncidentByNumber(
	ctx context.Context,
	runtimeConfig *SnowRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	config *models.ExecutionConfig,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("finally action time for ServiceNow", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate required fields
	if runtimeConfig.AccessToken == nil || *runtimeConfig.AccessToken == "" {
		return nil, fmt.Errorf("missing AccessToken for ServiceNow flow")
	}

	instanceURL := strings.Split(*config.CredentialBinding.AuthorityUrl, "/oauth_token.do")[0]

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	// Build request URL with query param
	number := inputs["number"].(string)
	url := fmt.Sprintf("%s/api/now/table/incident?sysparm_query=number=%s", instanceURL, number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call ServiceNow API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	// Decode ServiceNow response
	var snowResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&snowResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode ServiceNow response: %w", err),
		}, nil
	}

	// Check for ServiceNow result
	results, exists := snowResp["result"]
	if !exists {
		return &handler.ActionResult{
			Data:       snowResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: missing result object"),
		}, nil
	}

	resultArray, ok := results.([]interface{})
	if !ok || len(resultArray) == 0 {
		return &handler.ActionResult{
			Data:       snowResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: empty result array"),
		}, nil
	}

	incident := resultArray[0].(map[string]interface{})

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, incident, "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	} else {
		h.logger.Debug("Output validation successful")
	}

	// Check for ServiceNow result
	if _, exists := snowResp["result"]; !exists {
		return &handler.ActionResult{
			Data:       incident,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: missing result object"),
		}, nil
	}

	// Return ActionResult with full ServiceNow response
	return &handler.ActionResult{
		Data:       incident,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (h *SnowHandler) SearchIncidents(
	ctx context.Context,
	runtimeConfig *SnowRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	config *models.ExecutionConfig,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("finally action time for ServiceNow", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate required fields
	if runtimeConfig.AccessToken == nil || *runtimeConfig.AccessToken == "" {
		return nil, fmt.Errorf("missing AccessToken for ServiceNow flow")
	}

	instanceURL := strings.Split(*config.CredentialBinding.AuthorityUrl, "/oauth_token.do")[0]

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	// Build sysparm_query string from user input
	searchTerm := inputs["query"].(string)
	limit := 10 // default
	if l, ok := inputs["limit"].(int); ok && l > 0 {
		limit = l
	}

	// Construct full query including time filter
	sysparmQuery := fmt.Sprintf(
		"short_descriptionLIKE%s^ORdescriptionLIKE%s^opened_atONLast7days@javascript:gs.beginningOfLast7Days()",
		searchTerm,
		searchTerm,
	)

	params := url.Values{}
	params.Set("sysparm_query", sysparmQuery)
	params.Set("sysparm_limit", fmt.Sprintf("%d", limit))

	url := fmt.Sprintf("%s/api/now/table/incident?%s", instanceURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call ServiceNow API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	h.logger.Debug("raw response", zap.Any("raw", resp.Body))
	// Decode ServiceNow response
	var snowResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&snowResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode ServiceNow response: %w", err),
		}, nil
	}

	// Check for ServiceNow result
	results, exists := snowResp["result"]
	if !exists {
		return &handler.ActionResult{
			Data:       snowResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: missing result object"),
		}, nil
	}

	resultArray, ok := results.([]interface{})
	if !ok || len(resultArray) == 0 {
		return &handler.ActionResult{
			Data:       snowResp,
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("ServiceNow error: empty result array"),
		}, nil
	}

	// Validate outputs against schema for each incident
	for _, r := range resultArray {
		if incident, ok := r.(map[string]interface{}); ok {
			if err := commons.ValidateSchema(actionDef.OutputSchema, incident, "output"); err != nil {
				h.logger.Warn("Output validation warning",
					zap.String("action", actionDef.Name),
					zap.Error(err),
				)
			}
		}
	}
	h.logger.Debug("Output validation completed")

	// Return ActionResult with full ServiceNow response
	return &handler.ActionResult{
		Data:       snowResp,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}

func (h *SnowHandler) UpdateIncident(
	ctx context.Context,
	runtimeConfig *SnowRuntimeConfig,
	actionDef *models.ActionDefinition,
	binding models.IntegrationBinding,
	config *models.ExecutionConfig,
	inputs map[string]interface{},
) (*handler.ActionResult, error) {

	h.logger.Debug("finally action time for ServiceNow", zap.String("action", string(actionDef.Type)))
	commons.PrintCollectedParams(inputs)

	// Validate required fields
	if runtimeConfig.AccessToken == nil || *runtimeConfig.AccessToken == "" {
		return nil, fmt.Errorf("missing AccessToken for ServiceNow flow")
	}

	instanceURL := strings.Split(*config.CredentialBinding.AuthorityUrl, "/oauth_token.do")[0]

	// Validate inputs against schema
	if err := commons.ValidateSchema(actionDef.InputSchema, inputs, "input"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	h.logger.Debug("input validation successful")

	// Extract sys_id
	sysID, ok := inputs["sys_id"].(string)
	if !ok || sysID == "" {
		return nil, fmt.Errorf("missing or invalid sys_id in inputs")
	}

	// Prepare request payload (only include fields provided by user)
	payload := map[string]interface{}{}
	for key := range actionDef.InputSchema["properties"].(map[string]interface{}) {
		if val, exists := inputs[key]; exists && key != "sys_id" {
			payload[key] = val
		}
	}

	// Extract short description
	shortDesc, ok := inputs["short_description"].(string)
	if !ok || sysID == "" {
		return nil, fmt.Errorf("missing or invalid short_description in inputs")
	}

	payload["comments"] = shortDesc
	payload["work_notes"] = shortDesc

	body, err := json.Marshal(payload)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to marshal payload: %w", err),
		}, nil
	}

	// Build request
	url := fmt.Sprintf("%s/api/now/table/incident/%s", instanceURL, sysID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to create request: %w", err),
		}, nil
	}
	req.Header.Set("Authorization", "Bearer "+*runtimeConfig.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &handler.ActionResult{
			StatusCode: http.StatusInternalServerError,
			Error:      fmt.Errorf("failed to call ServiceNow API: %w", err),
		}, nil
	}
	defer resp.Body.Close()

	// Decode ServiceNow response
	var snowResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&snowResp); err != nil {
		return &handler.ActionResult{
			StatusCode: resp.StatusCode,
			Error:      fmt.Errorf("failed to decode ServiceNow response: %w", err),
		}, nil
	}

	// Validate outputs against schema
	if err := commons.ValidateSchema(actionDef.OutputSchema, snowResp["result"].(map[string]interface{}), "output"); err != nil {
		h.logger.Warn("Output validation warning",
			zap.String("action", actionDef.Name),
			zap.Error(err),
		)
	} else {
		h.logger.Debug("Output validation successful")
	}

	// Return ActionResult with full ServiceNow response
	return &handler.ActionResult{
		Data:       snowResp,
		StatusCode: resp.StatusCode,
		Metadata: map[string]interface{}{
			"action":  actionDef.Type,
			"binding": binding,
		},
	}, nil
}
