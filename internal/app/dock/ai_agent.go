package dock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	systemUserID    = "system"
	systemUsername  = "system"
	systemUserEmail = "system@local.polar"
)

type aiAgent struct {
	server       *Server
	apiKey       string
	baseURL      string
	model        string
	systemPrompt string
	tasks        chan aiAgentTask
	stopCh       chan struct{}
	stopOnce     sync.Once
	httpClient   *http.Client
}

type aiAgentTask struct {
	ThreadID int64
	UserID   string
	Content  string
}

type aiChatCompletionRequest struct {
	Model    string                    `json:"model"`
	Messages []aiChatCompletionMessage `json:"messages"`
}

type aiChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiChatCompletionResponse struct {
	Choices []struct {
		Message aiChatCompletionMessage `json:"message"`
	} `json:"choices"`
	Error json.RawMessage `json:"error,omitempty"`
}

func newAIAgent(server *Server, cfg Config) *aiAgent {
	if server == nil {
		return nil
	}
	return &aiAgent{
		server:       server,
		apiKey:       strings.TrimSpace(cfg.AIAgentAPIKey),
		baseURL:      strings.TrimSpace(cfg.AIAgentBaseURL),
		model:        strings.TrimSpace(cfg.AIAgentModel),
		systemPrompt: strings.TrimSpace(cfg.AIAgentSystemPrompt),
		tasks:        make(chan aiAgentTask, 64),
		stopCh:       make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (a *aiAgent) stop() {
	if a == nil {
		return
	}
	a.stopOnce.Do(func() {
		close(a.stopCh)
	})
}

func (a *aiAgent) enqueue(task aiAgentTask) {
	if a == nil || task.ThreadID <= 0 || task.UserID == "" || strings.TrimSpace(task.Content) == "" {
		return
	}
	select {
	case a.tasks <- task:
	default:
		log.Printf("ai agent queue full, dropping task for thread %d", task.ThreadID)
	}
}

func (a *aiAgent) run() {
	if a == nil {
		return
	}
	for {
		select {
		case <-a.stopCh:
			return
		case task := <-a.tasks:
			a.handleTask(task)
		}
	}
}

func (a *aiAgent) handleTask(task aiAgentTask) {
	reply, err := a.generateReply(task)
	if err != nil {
		log.Printf("ai agent reply failed: %v", err)
		reply = "我暂时无法完成这次处理，请稍后再试。"
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		reply = "我暂时没有可返回的结果。"
	}
	if _, err := a.server.sendChatMessage(task.ThreadID, systemUserID, systemUsername, reply, time.Now()); err != nil {
		log.Printf("send ai agent chat message failed: %v", err)
	}
}

func (a *aiAgent) generateReply(task aiAgentTask) (string, error) {
	if strings.TrimSpace(a.apiKey) == "" || strings.TrimSpace(a.baseURL) == "" || strings.TrimSpace(a.model) == "" {
		return "AI 助理尚未配置完成，请联系管理员设置 `AI_AGENT_API_KEY`、`AI_AGENT_BASE_URL` 和 `AI_AGENT_MODEL`。", nil
	}

	contextText, err := a.buildContext(task.ThreadID)
	if err != nil {
		log.Printf("build ai context failed: %v", err)
	}

	payload := aiChatCompletionRequest{
		Model: a.model,
		Messages: []aiChatCompletionMessage{
			{
				Role:    "system",
				Content: a.systemPrompt,
			},
			{
				Role:    "system",
				Content: contextText,
			},
			{
				Role:    "user",
				Content: task.Content,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	log.Printf("ai agent request: url=%s model=%s payload=%s", a.baseURL, a.model, compactLogJSON(body))

	req, err := http.NewRequest(http.MethodPost, a.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("ai agent response: status=%d body=%s", resp.StatusCode, compactLogJSON(responseBody))

	var result aiChatCompletionResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		if message := parseAIErrorMessage(result.Error); message != "" {
			return "", errors.New(message)
		}
		return "", fmt.Errorf("ai api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if len(result.Choices) == 0 {
		return "", errors.New("empty ai response")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func compactLogJSON(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		if normalized, err := json.Marshal(decoded); err == nil {
			text = string(normalized)
		}
	}

	const maxLogBytes = 4000
	if len(text) > maxLogBytes {
		return text[:maxLogBytes] + "...(truncated)"
	}
	return text
}

func parseAIErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

	var asObject struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(raw, &asObject); err == nil {
		if strings.TrimSpace(asObject.Message) != "" {
			return strings.TrimSpace(asObject.Message)
		}
		if strings.TrimSpace(asObject.Error) != "" {
			return strings.TrimSpace(asObject.Error)
		}
	}

	return strings.TrimSpace(string(raw))
}

func (a *aiAgent) buildContext(threadID int64) (string, error) {
	var parts []string
	parts = append(parts, "以下是程序运行目录中的文档摘要和当前私聊上下文。")

	docText, err := a.loadRuntimeDocuments()
	if err != nil {
		parts = append(parts, "文档读取失败："+err.Error())
	} else if docText != "" {
		parts = append(parts, docText)
	}

	messages, err := a.server.listRecentChatMessages(threadID, 12)
	if err != nil {
		return strings.Join(parts, "\n\n"), err
	}
	if len(messages) > 0 {
		var b strings.Builder
		b.WriteString("最近会话消息：\n")
		for _, msg := range messages {
			if msg.Deleted {
				continue
			}
			name := msg.SenderUsername
			if name == "" {
				name = msg.SenderID
			}
			b.WriteString("- ")
			b.WriteString(name)
			b.WriteString(": ")
			b.WriteString(strings.TrimSpace(msg.Content))
			b.WriteString("\n")
		}
		parts = append(parts, b.String())
	}

	return strings.Join(parts, "\n\n"), nil
}

func (a *aiAgent) loadRuntimeDocuments() (string, error) {
	root := strings.TrimSpace(a.server.workDir)
	if root == "" {
		return "", nil
	}

	maxFiles := 8
	maxBytes := 24 * 1024
	used := 0
	collected := 0
	var b strings.Builder

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == "dist" || base == ".gocache" || base == "data" {
				return filepath.SkipDir
			}
			return nil
		}
		if collected >= maxFiles || used >= maxBytes {
			return fs.SkipAll
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := strings.TrimSpace(string(content))
		if text == "" {
			return nil
		}
		remaining := maxBytes - used
		if remaining <= 0 {
			return fs.SkipAll
		}
		if len(text) > remaining {
			text = text[:remaining]
		}
		b.WriteString("文件：")
		b.WriteString(rel)
		b.WriteString("\n")
		b.WriteString(text)
		b.WriteString("\n\n")
		used += len(text)
		collected++
		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil && !errors.Is(err, fs.SkipAll) {
		return "", err
	}
	return strings.TrimSpace(b.String()), nil
}
