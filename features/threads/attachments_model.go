package threads

import (
	"database/sql"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"strings"
	"time"
)

type Attachment struct {
	AttachmentID int       `json:"attachmentId"`
	BotID        int       `json:"botId"`
	MessageID    *int      `json:"messageId"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	Size         int       `json:"size"`
	StoredName   string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

const attachmentColumns = "attachment_id, bot_id, message_id, filename, mime_type, size, path, created_at"

func scanAttachment(row interface{ Scan(...any) error }) (Attachment, error) {
	var attachment Attachment
	err := row.Scan(&attachment.AttachmentID, &attachment.BotID, &attachment.MessageID, &attachment.Filename, &attachment.MimeType, &attachment.Size, &attachment.StoredName, &attachment.CreatedAt)
	return attachment, err
}

func CreateAttachment(attachment Attachment) (Attachment, error) {
	query := "INSERT INTO attachments (bot_id, filename, mime_type, size, path) VALUES (?, ?, ?, ?, ?) RETURNING " + attachmentColumns

	created, err := scanAttachment(sqlite.DB.QueryRow(query, attachment.BotID, attachment.Filename, attachment.MimeType, attachment.Size, attachment.StoredName))
	if err != nil {
		return created, fmt.Errorf("error creating attachment: %w", err)
	}

	return created, nil
}

func GetAttachment(attachmentID int) (Attachment, error) {
	attachment, err := scanAttachment(sqlite.DB.QueryRow("SELECT "+attachmentColumns+" FROM attachments WHERE attachment_id = ?", attachmentID))
	if err != nil {
		return attachment, fmt.Errorf("error retrieving attachment %d: %w", attachmentID, utils.NotFoundIfNoRows(err))
	}

	return attachment, nil
}

// LinkAttachmentsTx only claims uploads from the same bot that aren't on a message yet, so an id can't be reused across threads.
func LinkAttachmentsTx(tx *sql.Tx, botID, messageID int, attachmentIDs []int) error {
	for _, attachmentID := range attachmentIDs {
		query := "UPDATE attachments SET message_id = ? WHERE attachment_id = ? AND bot_id = ? AND message_id IS NULL"

		err := utils.RequireAffected(tx.Exec(query, messageID, attachmentID, botID))
		if err != nil {
			return fmt.Errorf("error linking attachment %d: %w", attachmentID, err)
		}
	}

	return nil
}

func getMessageAttachmentsTx(tx *sql.Tx, messageID int) ([]Attachment, error) {
	attachments := []Attachment{}

	rows, err := tx.Query("SELECT "+attachmentColumns+" FROM attachments WHERE message_id = ? ORDER BY attachment_id", messageID)
	if err != nil {
		return nil, fmt.Errorf("error reading message %d attachments: %w", messageID, err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning message %d attachment: %w", messageID, err)
		}
		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message %d attachments: %w", messageID, err)
	}

	return attachments, nil
}

func GetThreadAttachments(threadID int) ([]Attachment, error) {
	attachments := []Attachment{}

	query := `
		SELECT a.attachment_id, a.bot_id, a.message_id, a.filename, a.mime_type, a.size, a.path, a.created_at
		FROM attachments a
		JOIN messages m ON m.message_id = a.message_id
		WHERE m.thread_id = ?
	`

	rows, err := sqlite.DB.Query(query, threadID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving thread attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning attachment: %w", err)
		}
		attachments = append(attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attachments: %w", err)
	}

	return attachments, nil
}

func addAttachments(messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	messageIndexes := map[int]int{}
	placeholders := []string{}
	args := []any{}
	for i, message := range messages {
		messageIndexes[message.MessageID] = i
		placeholders = append(placeholders, "?")
		args = append(args, message.MessageID)
	}

	query := "SELECT " + attachmentColumns + " FROM attachments WHERE message_id IN (" + strings.Join(placeholders, ",") + ") ORDER BY attachment_id"

	rows, err := sqlite.DB.Query(query, args...)
	if err != nil {
		return fmt.Errorf("error retrieving attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment, err := scanAttachment(rows)
		if err != nil {
			return fmt.Errorf("error scanning attachment: %w", err)
		}

		index := messageIndexes[*attachment.MessageID]
		messages[index].Attachments = append(messages[index].Attachments, attachment)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating attachments: %w", err)
	}

	return nil
}
