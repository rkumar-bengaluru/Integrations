package handler

import "agent.fabric.com/modules/internal/models"

// ActionMenuItem represents a menu item for an action
type ActionMenuItem struct {
	Action  models.ActionDefinition
	Display string
	Number  int
}
