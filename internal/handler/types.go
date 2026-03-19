package handler

import "github.com/rkumar-bengaluru/Integrations/v2/internal/models"

// ActionMenuItem represents a menu item for an action
type ActionMenuItem struct {
	Action  models.ActionDefinition
	Display string
	Number  int
}
