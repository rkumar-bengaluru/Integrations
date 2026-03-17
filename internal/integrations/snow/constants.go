package snow

var (
	SnowOauth2ClientCredentialFlowName            = "Symphony Platform Bot Snow App Credentials"
	SnowOauth2AuthorizationCodeFlowCredentialName = "Symphony Platform User Slack Credentials"

	SnowVendorName      = "snow"
	SnowIntegrationName = "ServiceNow Integration"

	// test action
	SnowTestActionName = "TestAction"
	SnowTestActionType = "test_action"

	// create incident
	SnowCreateIncidentActionName = "Create a New Incident in Snow"
	SnowCreateIncidentActionType = "create_incident"
	// get incident by number
	SnowGetIncidentActionName = "Get Incident by Number"
	SnowGetIncidentActionType = "servicenow_get_incident"
	// search incident
	SnowSearchIncidentActionName = "Search Existing Tickets"
	SnowSearchIncidentActionType = "servicenow_search_incident"
	// update incident
	SnowUpdateIncidentActionName = "Update Incident"
	SnowUpdateIncidentActionType = "servicenow_update_incident"
)
