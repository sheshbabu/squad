package threads

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"squad/commons/utils"
	"squad/features/bots"
	"strconv"
)

const maxAttachmentSize = 10 << 20

var attachmentExtensions = map[string]string{
	"image/png":       ".png",
	"image/jpeg":      ".jpg",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"application/pdf": ".pdf",
}

func GetAttachmentPath(attachment Attachment) string {
	return filepath.Join(bots.GetBotFolderPath(attachment.BotID), "attachments", attachment.StoredName)
}

func HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	if _, err := bots.GetBot(botID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_READ_FAILED", "Failed to load bot", err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+(1<<20))

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FILE", "Files must be 10 MB or smaller", nil)
		return
	}
	defer file.Close()

	if header.Size > maxAttachmentSize {
		utils.SendErrorResponse(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Files must be 10 MB or smaller", nil)
		return
	}

	// The browser's content type is a guess from the extension, sniffing the bytes is what the model will actually receive.
	head := make([]byte, 512)
	headLength, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FILE", "Failed to read file", nil)
		return
	}
	mimeType := http.DetectContentType(head[:headLength])

	extension, ok := attachmentExtensions[mimeType]
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "UNSUPPORTED_FILE", "Only images and PDFs can be attached", nil)
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "ATTACHMENT_SAVE_FAILED", "Failed to save file", fmt.Errorf("error rewinding upload: %w", err))
		return
	}

	storedName, err := saveAttachmentFile(botID, extension, file)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "ATTACHMENT_SAVE_FAILED", "Failed to save file", err)
		return
	}

	attachment, err := CreateAttachment(Attachment{BotID: botID, Filename: filepath.Base(header.Filename), MimeType: mimeType, Size: int(header.Size), StoredName: storedName})
	if err != nil {
		os.Remove(filepath.Join(bots.GetBotFolderPath(botID), "attachments", storedName))
		utils.SendErrorResponse(w, http.StatusInternalServerError, "ATTACHMENT_SAVE_FAILED", "Failed to save file", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, attachment)
}

func HandleGetAttachment(w http.ResponseWriter, r *http.Request) {
	attachmentID, err := strconv.Atoi(r.PathValue("attachmentId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_ATTACHMENT_ID", "Invalid attachment", nil)
		return
	}

	attachment, err := GetAttachment(attachmentID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "ATTACHMENT_READ_FAILED", "Failed to load attachment", err)
		return
	}

	w.Header().Set("Content-Type", attachment.MimeType)
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeFile(w, r, GetAttachmentPath(attachment))
}

func saveAttachmentFile(botID int, extension string, source io.Reader) (string, error) {
	folder := filepath.Join(bots.GetBotFolderPath(botID), "attachments")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return "", fmt.Errorf("error creating attachments folder: %w", err)
	}

	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("error generating attachment name: %w", err)
	}
	storedName := hex.EncodeToString(randomBytes) + extension

	destination, err := os.Create(filepath.Join(folder, storedName))
	if err != nil {
		return "", fmt.Errorf("error creating attachment file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return "", fmt.Errorf("error writing attachment file: %w", err)
	}

	return storedName, nil
}
