package dock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) saveMarkdownDocument(userID, title, content string, isPublic bool, now time.Time) (*MarkdownEntry, string, error) {
	if err := os.MkdirAll(s.markdownDir, 0o755); err != nil {
		return nil, "", err
	}

	safeTitle := sanitizeFilename(title)
	timestamp := now.Format("20060102_150405")
	filename := safeTitle + "_" + timestamp + "_" + sanitizeFilename(userID) + ".md"
	path := filepath.Join(s.markdownDir, filename)

	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		content = "# " + title + "\n\n" + content
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, "", err
	}

	entryID, err := s.createMarkdownEntryReturningID(userID, title, path, isPublic, now)
	if err != nil {
		_ = os.Remove(path)
		return nil, "", err
	}

	return &MarkdownEntry{
		ID:         entryID,
		UserID:     userID,
		Title:      title,
		FilePath:   path,
		IsPublic:   isPublic,
		UploadedAt: now,
	}, content, nil
}

func buildSystemMarkdownTitle(content string, now time.Time) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#")
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 60 {
				line = line[:60]
			}
			return line
		}
	}
	return fmt.Sprintf("system-reply-%s", now.Format("20060102-150405"))
}

func buildMarkdownPreview(content string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 80
	}
	replacer := strings.NewReplacer(
		"\r", " ",
		"\n", " ",
		"`", "",
		"#", "",
		"*", "",
		">", "",
		"[", "",
		"]", "",
		"(", "",
		")", "",
	)
	text := strings.TrimSpace(replacer.Replace(content))
	text = strings.Join(strings.Fields(text), " ")
	if text == "" {
		return "AI 文档回复"
	}
	if len(text) > maxLength {
		return text[:maxLength] + "..."
	}
	return text
}
