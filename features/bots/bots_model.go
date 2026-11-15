package bots

import (
	"database/sql"
	"errors"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"strings"
	"time"
)

type Bot struct {
	BotID        int    `json:"botId"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Color        string `json:"color"`
	SystemPrompt string `json:"systemPrompt"`
	Gateway      string `json:"gateway"`
	Model        string `json:"model"`
	Provider     string `json:"provider"`
	// Groups are stored as turned off rather than on, so every available tool starts enabled, including ones added later.
	DisabledToolGroups []string  `json:"disabledToolGroups"`
	ZenAPIToken        string    `json:"zenApiToken"`
	WebhookToken       string    `json:"webhookToken"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

const botColumns = "bot_id, name, icon, color, system_prompt, gateway, model, provider, disabled_tool_groups, zen_api_token, webhook_token, created_at, updated_at"

func scanBot(row interface{ Scan(...any) error }) (Bot, error) {
	var bot Bot
	var disabledToolGroups string
	err := row.Scan(&bot.BotID, &bot.Name, &bot.Icon, &bot.Color, &bot.SystemPrompt, &bot.Gateway, &bot.Model, &bot.Provider, &disabledToolGroups, &bot.ZenAPIToken, &bot.WebhookToken, &bot.CreatedAt, &bot.UpdatedAt)
	bot.DisabledToolGroups = parseToolGroups(disabledToolGroups)
	return bot, err
}

func parseToolGroups(value string) []string {
	groups := []string{}
	for _, group := range strings.Split(value, ",") {
		if group != "" {
			groups = append(groups, group)
		}
	}
	return groups
}

func GetBots() ([]Bot, error) {
	bots := []Bot{}

	rows, err := sqlite.DB.Query("SELECT " + botColumns + " FROM bots ORDER BY name COLLATE NOCASE")
	if err != nil {
		return nil, fmt.Errorf("error retrieving bots: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		bot, err := scanBot(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning bot: %w", err)
		}
		bots = append(bots, bot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bots: %w", err)
	}

	return bots, nil
}

func GetBot(botID int) (Bot, error) {
	bot, err := scanBot(sqlite.DB.QueryRow("SELECT "+botColumns+" FROM bots WHERE bot_id = ?", botID))
	if errors.Is(err, sql.ErrNoRows) {
		return bot, fmt.Errorf("error retrieving bot %d: %w", botID, utils.ErrNotFound)
	}
	if err != nil {
		return bot, fmt.Errorf("error retrieving bot %d: %w", botID, err)
	}

	return bot, nil
}

func CreateBot(bot Bot) (Bot, error) {
	query := `
		INSERT INTO bots (name, icon, color, system_prompt, gateway, model, provider)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING ` + botColumns

	created, err := scanBot(sqlite.DB.QueryRow(query, bot.Name, bot.Icon, bot.Color, bot.SystemPrompt, bot.Gateway, bot.Model, bot.Provider))
	if err != nil {
		return created, fmt.Errorf("error creating bot: %w", err)
	}

	return created, nil
}

func UpdateBot(bot Bot) (Bot, error) {
	query := `
		UPDATE bots
		SET name = ?, icon = ?, color = ?, system_prompt = ?, gateway = ?, model = ?, provider = ?, updated_at = CURRENT_TIMESTAMP
		WHERE bot_id = ?
		RETURNING ` + botColumns

	updated, err := scanBot(sqlite.DB.QueryRow(query, bot.Name, bot.Icon, bot.Color, bot.SystemPrompt, bot.Gateway, bot.Model, bot.Provider, bot.BotID))
	if errors.Is(err, sql.ErrNoRows) {
		return updated, fmt.Errorf("error updating bot %d: %w", bot.BotID, utils.ErrNotFound)
	}
	if err != nil {
		return updated, fmt.Errorf("error updating bot %d: %w", bot.BotID, err)
	}

	return updated, nil
}

func UpdateBotTools(botID int, disabledToolGroups []string, zenAPIToken string) (Bot, error) {
	query := `
		UPDATE bots
		SET disabled_tool_groups = ?, zen_api_token = ?, updated_at = CURRENT_TIMESTAMP
		WHERE bot_id = ?
		RETURNING ` + botColumns

	updated, err := scanBot(sqlite.DB.QueryRow(query, strings.Join(disabledToolGroups, ","), zenAPIToken, botID))
	if errors.Is(err, sql.ErrNoRows) {
		return updated, fmt.Errorf("error updating bot %d tools: %w", botID, utils.ErrNotFound)
	}
	if err != nil {
		return updated, fmt.Errorf("error updating bot %d tools: %w", botID, err)
	}

	return updated, nil
}

func UpdateBotWebhookToken(botID int, webhookToken string) (Bot, error) {
	query := `
		UPDATE bots
		SET webhook_token = ?, updated_at = CURRENT_TIMESTAMP
		WHERE bot_id = ?
		RETURNING ` + botColumns

	updated, err := scanBot(sqlite.DB.QueryRow(query, webhookToken, botID))
	if errors.Is(err, sql.ErrNoRows) {
		return updated, fmt.Errorf("error updating bot %d webhook: %w", botID, utils.ErrNotFound)
	}
	if err != nil {
		return updated, fmt.Errorf("error updating bot %d webhook: %w", botID, err)
	}

	return updated, nil
}

func GetBotByWebhookToken(webhookToken string) (Bot, error) {
	bot, err := scanBot(sqlite.DB.QueryRow("SELECT "+botColumns+" FROM bots WHERE webhook_token = ? AND webhook_token != ''", webhookToken))
	if errors.Is(err, sql.ErrNoRows) {
		return bot, fmt.Errorf("error getting bot by webhook token: %w", utils.ErrNotFound)
	}
	if err != nil {
		return bot, fmt.Errorf("error getting bot by webhook token: %w", err)
	}

	return bot, nil
}

func DeleteBot(botID int) error {
	result, err := sqlite.DB.Exec("DELETE FROM bots WHERE bot_id = ?", botID)
	if err != nil {
		return fmt.Errorf("error deleting bot %d: %w", botID, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted bot %d: %w", botID, err)
	}
	if count == 0 {
		return fmt.Errorf("error deleting bot %d: %w", botID, utils.ErrNotFound)
	}

	return nil
}
