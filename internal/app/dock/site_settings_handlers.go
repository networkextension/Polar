package dock

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func normalizeApplePushEnvironment(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dev":
		return "dev", nil
	case "prod":
		return "prod", nil
	default:
		return "", errors.New("invalid apple push certificate environment")
	}
}

func isAllowedApplePushCertificate(filename string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(filename))) {
	case ".p8", ".p12", ".pem", ".cer", ".crt", ".key":
		return true
	default:
		return false
	}
}

func (s *Server) applePushCertificateDir() string {
	return filepath.Join(s.uploadDir, "apple_push")
}

func applePushCertificateFromSettings(settings *SiteSettings, environment string) *ApplePushCertificate {
	if settings == nil {
		return nil
	}
	if environment == "dev" {
		return settings.ApplePushDevCert
	}
	return settings.ApplePushProdCert
}

func (s *Server) handleSiteSettingsGet(c *gin.Context) {
	settings, err := s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"site": s.hydrateSiteSettings(settings),
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
		"site":    s.hydrateSiteSettings(settings),
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
		"site":     s.hydrateSiteSettings(settings),
	})
}

func (s *Server) handleApplePushCertificateUpload(c *gin.Context) {
	if s.uploadDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传目录未配置"})
		return
	}

	environment, err := normalizeApplePushEnvironment(c.Query("env"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "环境参数无效，仅支持 dev 或 prod"})
		return
	}

	file, err := c.FormFile("certificate")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择证书文件"})
		return
	}

	if !isAllowedApplePushCertificate(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .p8、.p12、.pem、.cer、.crt、.key 证书文件"})
		return
	}

	certDir := s.applePushCertificateDir()
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	filename := "apple_push_" + environment + "_" + buildUploadFilename(file.Filename)
	dstPath := filepath.Join(certDir, filename)
	if err := c.SaveUploadedFile(file, dstPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	fileURL := "/uploads/apple_push/" + filename
	now := time.Now()
	settings, _ := s.getSiteSettings()
	oldCert := applePushCertificateFromSettings(settings, environment)
	if err := s.updateApplePushCertificate(environment, fileURL, file.Filename, now); err != nil {
		_ = os.Remove(dstPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	if oldCert != nil && oldCert.FileURL != "" && strings.HasPrefix(oldCert.FileURL, "/uploads/apple_push/") {
		oldPath := filepath.Join(certDir, filepath.Base(oldCert.FileURL))
		if oldPath != dstPath {
			_ = os.Remove(oldPath)
		}
	}

	settings, err = s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Apple Push 证书已上传",
		"environment": environment,
		"site":        s.hydrateSiteSettings(settings),
	})
}

func (s *Server) handleApplePushCertificateDelete(c *gin.Context) {
	environment, err := normalizeApplePushEnvironment(c.Query("env"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "环境参数无效，仅支持 dev 或 prod"})
		return
	}

	settings, err := s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	cert := applePushCertificateFromSettings(settings, environment)
	if cert == nil || cert.FileURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "当前环境下没有已上传的证书"})
		return
	}

	if err := s.clearApplePushCertificate(environment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	if s.uploadDir != "" && strings.HasPrefix(cert.FileURL, "/uploads/apple_push/") {
		_ = os.Remove(filepath.Join(s.applePushCertificateDir(), filepath.Base(cert.FileURL)))
	}

	settings, err = s.getSiteSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Apple Push 证书已删除",
		"environment": environment,
		"site":        s.hydrateSiteSettings(settings),
	})
}
