package main

import (
	"context"
	"fmt"
	"os"

	"agent.fabric.com/modules/internal/integrations/slack"
	"agent.fabric.com/modules/internal/logger"
	"agent.fabric.com/modules/internal/repository"
	"agent.fabric.com/modules/internal/repository/db"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()

	// Load .env file into environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: Error loading .env file:", err)
		panic(err)
	}

	// Initialize services
	keyID := os.Getenv("ENCRYPTION_KEY_ID")
	if keyID == "" {
		keyID = "default-key-v1"
	}

	// Tenant ID
	tenantID := os.Getenv("SYMPHONY_TENANT_ID")
	if tenantID == "" {
		tenantID = "default-key-v1"
	}

	// Tenant and Integration IDs for test
	tenantUID := uuid.MustParse(tenantID)

	// logger
	logger := logger.Get(ctx)

	conn, dialect := db.CreateDB(ctx, "genei-server")
	defer conn.Close()
	_, database := repository.NewSQLStore(dialect, logger)

	action, err := slack.AddListUsersAction(ctx, database, logger, tenantUID,
		slack.SlackListUsersActionName,
		slack.SlackListUsersActionType,
		slack.SlackIntegrationName)
	if err != nil {
		fmt.Errorf("error creating action...%w", err)
	}

	fmt.Println(fmt.Sprintf("action %s created for integration %s", action.Name, slack.SlackIntegrationName))

	action, err = slack.AddInviteToChannelAction(ctx, database, logger, tenantUID,
		slack.SlackInviteUsersActionName,
		slack.SlackInviteUsersActionType,
		slack.SlackIntegrationName)
	if err != nil {
		fmt.Errorf("error creating action...%w", err)
	}

	fmt.Println(fmt.Sprintf("action %s created for integration %s", action.Name, slack.SlackIntegrationName))

	action, err = slack.AddListChannelsAction(ctx, database, logger, tenantUID,
		slack.SlackListChannelsActionName,
		slack.SlackListChannelsActionType,
		slack.SlackIntegrationName)
	if err != nil {
		fmt.Errorf("error creating action...%w", err)
	}

	fmt.Println(fmt.Sprintf("action %s created for integration %s", action.Name, slack.SlackIntegrationName))

	action, err = slack.AddPostMessageToChannelAction(ctx, database, logger, tenantUID,
		slack.SlackPostMessageActionName,
		slack.SlackPostMessageActionType,
		slack.SlackIntegrationName)
	if err != nil {
		fmt.Errorf("error creating action...%w", err)
	}

	fmt.Println(fmt.Sprintf("action %s created for integration %s", action.Name, slack.SlackIntegrationName))

}
