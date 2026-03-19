package dock

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxIconSizeBytes = 2 << 20

func (s *Server) handleUserIconUpload(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

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

	filename := "icon_" + sanitizeFilename(userIDStr) + "_" + buildUploadFilename(file.Filename)
	dstPath := filepath.Join(s.uploadDir, filename)
	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	iconURL := "/uploads/" + filename

	var oldIcon string
	if user, err := s.getUserByID(userIDStr); err == nil && user != nil {
		oldIcon = user.IconURL
	}
	if oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/") {
		oldPath := filepath.Join(s.uploadDir, filepath.Base(oldIcon))
		if oldPath != dstPath {
			_ = os.Remove(oldPath)
		}
	}

	if err := s.updateUserIcon(userIDStr, iconURL); err != nil {
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "更新成功",
		"icon_url": iconURL,
	})
}
