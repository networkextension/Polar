package dock

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleTaskResultsList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务"})
		return
	}

	results, err := s.listTaskResults(postID, userIDStr)
	if err != nil {
		switch {
		case errors.Is(err, errTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		case errors.Is(err, errTaskForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "仅发布者和执行者可查看任务成果"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *Server) handleTaskResultCreate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务"})
		return
	}

	canView, canSubmit, err := s.canAccessTaskResults(postID, userIDStr)
	if err != nil {
		switch {
		case errors.Is(err, errTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		}
		return
	}
	if !canView || !canSubmit {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅被选中的执行者可提交任务成果"})
		return
	}

	note := strings.TrimSpace(c.PostForm("note"))
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的表单数据"})
		return
	}

	imageFiles := form.File["images"]
	videoFiles := form.File["videos"]
	if len(imageFiles) == 0 && len(videoFiles) == 0 && note == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少上传图片、视频或填写说明"})
		return
	}

	for _, file := range imageFiles {
		if file == nil {
			continue
		}
		if !isUploadType(file, "image/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持上传图片文件"})
			return
		}
	}
	for _, file := range videoFiles {
		if file == nil {
			continue
		}
		if !isUploadType(file, "video/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持上传视频文件"})
			return
		}
	}

	if s.uploadDir == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传目录未配置"})
		return
	}
	if err := os.MkdirAll(s.uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	now := time.Now()
	resultID, err := s.createTaskResult(postID, userIDStr, note, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	imageURLs := make([]string, 0, len(imageFiles))
	videoURLs := make([]string, 0, len(videoFiles))
	videoItems := make([]PostVideo, 0, len(videoFiles))

	for _, file := range imageFiles {
		if file == nil {
			continue
		}
		filename := buildUploadFilename(file.Filename)
		dstPath := filepath.Join(s.uploadDir, filename)
		if err := c.SaveUploadedFile(file, dstPath); err != nil {
			_ = s.deleteTaskResult(resultID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
			return
		}
		publicURL := "/uploads/" + filename
		if err := s.addTaskResultImage(resultID, publicURL, now); err != nil {
			_ = s.deleteTaskResult(resultID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
			return
		}
		imageURLs = append(imageURLs, publicURL)
	}

	for _, file := range videoFiles {
		if file == nil {
			continue
		}
		filename := buildUploadFilename(file.Filename)
		dstPath := filepath.Join(s.uploadDir, filename)
		if err := c.SaveUploadedFile(file, dstPath); err != nil {
			_ = s.deleteTaskResult(resultID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "视频保存失败"})
			return
		}

		publicURL := "/uploads/" + filename
		posterURL := ""
		posterFilename := buildDerivedUploadFilename(filename, "poster", ".jpg")
		posterPath := filepath.Join(s.uploadDir, posterFilename)
		posterPublicURL := "/uploads/" + posterFilename

		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		err := generateVideoPoster(ctx, dstPath, posterPath)
		cancel()
		if err != nil {
			log.Printf("generate task result poster failed for %s: %v", dstPath, err)
		} else {
			posterURL = posterPublicURL
		}

		if err := s.addTaskResultVideo(resultID, publicURL, posterURL, now); err != nil {
			_ = s.deleteTaskResult(resultID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
			return
		}
		videoURLs = append(videoURLs, publicURL)
		videoItems = append(videoItems, PostVideo{URL: publicURL, PosterURL: posterURL})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "任务成果已提交",
		"id":          resultID,
		"note":        note,
		"images":      imageURLs,
		"videos":      videoURLs,
		"video_items": videoItems,
		"created_at":  now,
	})
}
