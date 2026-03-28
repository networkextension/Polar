package dock

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) packTunnelRulesDir() string {
	if strings.TrimSpace(s.uploadDir) == "" {
		return ""
	}
	return filepath.Join(s.uploadDir, "packtunnel_rules")
}

func (s *Server) handlePackTunnelRuleUpload(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel rules upload ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if s.uploadDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传目录未配置"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 rules 文件"})
		return
	}

	rulesDir := s.packTunnelRulesDir()
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" || len(ext) > 16 {
		ext = ".txt"
	}
	storedName := "packtunnel_rules_global_" + buildUploadFilename("rules"+ext)
	dstPath := filepath.Join(rulesDir, storedName)
	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	existing, err := s.getPackTunnelRuleFile(systemUserID)
	if err != nil {
		log.Printf("packtunnel rules get existing failed: %v", err)
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	now := time.Now()
	item := PackTunnelRuleFile{
		UserID:      systemUserID,
		FileName:    file.Filename,
		StoredName:  storedName,
		FilePath:    dstPath,
		Size:        file.Size,
		ContentType: contentType,
		UploadedAt:  now,
	}
	if err := s.upsertPackTunnelRuleFile(item); err != nil {
		log.Printf("packtunnel rules upsert failed filename=%s: %v", file.Filename, err)
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	if existing != nil && existing.FilePath != "" && existing.FilePath != dstPath {
		_ = os.Remove(existing.FilePath)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "rules 已上传",
		"rule":    item,
	})
}

func (s *Server) handlePackTunnelRuleDownload(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel rules download ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	item, err := s.getPackTunnelRuleFile(systemUserID)
	if err != nil {
		log.Printf("packtunnel rules download lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rules 文件不存在"})
		return
	}
	if _, err := os.Stat(item.FilePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rules 文件不存在"})
		return
	}

	if item.ContentType != "" {
		c.Header("Content-Type", item.ContentType)
	}
	c.FileAttachment(item.FilePath, item.FileName)
}

func (s *Server) handlePackTunnelRuleDelete(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel rules delete ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	item, err := s.getPackTunnelRuleFile(systemUserID)
	if err != nil {
		log.Printf("packtunnel rules delete lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "rules 文件不存在"})
		return
	}

	if _, err := os.Stat(item.FilePath); err == nil {
		_ = os.Remove(item.FilePath)
	}
	ok, err := s.deletePackTunnelRuleFile(systemUserID)
	if err != nil {
		log.Printf("packtunnel rules delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "rules 文件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "rules 已删除"})
}
