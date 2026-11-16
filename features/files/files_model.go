package files

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"squad/commons/utils"
	"squad/features/bots"
	"strings"
	"time"
)

type File struct {
	Name      string    `json:"name"`
	Title     string    `json:"title"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FileContent struct {
	File
	Content string `json:"content"`
}

var ErrInvalidName = errors.New("file names must end in .md and use only letters, numbers, spaces, dots, dashes and underscores")

var fileNamePattern = regexp.MustCompile(`^[A-Za-z0-9 _.-]+\.md$`)

func ValidateName(name string) error {
	if !fileNamePattern.MatchString(name) || strings.HasPrefix(name, ".") {
		return ErrInvalidName
	}
	return nil
}

// openFolder goes through os.Root so no name, however crafted, can reach outside the bot's folder.
func openFolder(botID int) (*os.Root, error) {
	folder := filepath.Join(bots.GetBotFolderPath(botID), "files")
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("error creating files folder: %w", err)
	}

	root, err := os.OpenRoot(folder)
	if err != nil {
		return nil, fmt.Errorf("error opening files folder: %w", err)
	}

	return root, nil
}

func ListFiles(botID int) ([]File, error) {
	root, err := openFolder(botID)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	files := []File{}
	for _, entry := range entries {
		if entry.IsDir() || ValidateName(entry.Name()) != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("error reading file info: %w", err)
		}

		files = append(files, File{Name: entry.Name(), Title: readTitle(root, entry.Name()), Size: info.Size(), UpdatedAt: info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

func readTitle(root *os.Root, name string) string {
	file, err := root.Open(name)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(scanner.Text()), "#"))
		if line != "" {
			return line
		}
	}

	return ""
}

func ReadFile(botID int, name string) (FileContent, error) {
	var fileContent FileContent

	if err := ValidateName(name); err != nil {
		return fileContent, err
	}

	root, err := openFolder(botID)
	if err != nil {
		return fileContent, err
	}
	defer root.Close()

	file, err := root.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return fileContent, fmt.Errorf("error reading %s: %w", name, utils.ErrNotFound)
	}
	if err != nil {
		return fileContent, fmt.Errorf("error opening %s: %w", name, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fileContent, fmt.Errorf("error reading %s info: %w", name, err)
	}

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return fileContent, fmt.Errorf("error reading %s: %w", name, err)
	}

	fileContent.File = File{Name: name, Size: info.Size(), UpdatedAt: info.ModTime()}
	fileContent.Content = string(contentBytes)

	return fileContent, nil
}

func WriteFile(botID int, name, content string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	root, err := openFolder(botID)
	if err != nil {
		return err
	}
	defer root.Close()

	file, err := root.Create(name)
	if err != nil {
		return fmt.Errorf("error creating %s: %w", name, err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("error writing %s: %w", name, err)
	}

	return nil
}

// EditFile replaces one exact occurrence, so the bot must quote enough context to be unambiguous instead of rewriting the whole file.
func EditFile(botID int, name, oldText, newText string) error {
	fileContent, err := ReadFile(botID, name)
	if err != nil {
		return err
	}

	count := strings.Count(fileContent.Content, oldText)
	if oldText == "" || count == 0 {
		return fmt.Errorf("the text to replace was not found in %s", name)
	}
	if count > 1 {
		return fmt.Errorf("the text to replace appears %d times in %s, include more surrounding text so it matches once", count, name)
	}

	return WriteFile(botID, name, strings.Replace(fileContent.Content, oldText, newText, 1))
}

func DeleteFile(botID int, name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	root, err := openFolder(botID)
	if err != nil {
		return err
	}
	defer root.Close()

	err = root.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("error deleting %s: %w", name, utils.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("error deleting %s: %w", name, err)
	}

	return nil
}
