package dock

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleSiteSettingsGet(c *gin.Context) {
	settings, err := s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"site": settings,
	})
}

func (s *Server) handleSiteSettingsUpdate(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "站点名称不能为空"})
		return
	}

	if err := s.updateSiteSettings(name, strings.TrimSpace(req.Description)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	settings, err := s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "保存成功",
		"site":    settings,
	})
}

func (s *Server) handleSiteIconUpload(c *gin.Context) {
	if s.uploadDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传目录未配置"})
		return
	}

	file, err := c.FormFile("icon")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}

	if file.Size > maxIconSizeBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片过大，建议不超过 2MB"})
		return
	}

	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持图片格式"})
		return
	}

	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	filename := "site_icon_" + buildUploadFilename(file.Filename)
	dstPath := filepath.Join(s.uploadDir, filename)
	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	iconURL := "/uploads/" + filename

	var oldIcon string
	if settings, err := s.getSiteSettings(); err == nil && settings != nil {
		oldIcon = settings.IconURL
	}
	if oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/") {
		oldPath := filepath.Join(s.uploadDir, filepath.Base(oldIcon))
		if oldPath != dstPath {
			_ = os.Remove(oldPath)
		}
	}

	if err := s.updateSiteIcon(iconURL); err != nil {
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	settings, err := s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "更新成功",
		"icon_url": iconURL,
		"site":     settings,
	})
}
