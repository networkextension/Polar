package dock

import (
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handlePostCreate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	content := strings.TrimSpace(c.PostForm("content"))
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "内容不能为空"})
		return
	}

	var tagID *int64
	if tagIDStr := strings.TrimSpace(c.PostForm("tag_id")); tagIDStr != "" {
		parsed, err := strconv.ParseInt(tagIDStr, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的标签"})
			return
		}
		tagID = &parsed
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的表单数据"})
		return
	}

	files := form.File["images"]
	videoFiles := form.File["videos"]
	for _, file := range files {
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
	postID, err := s.createPost(userIDStr, tagID, content, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	savedFiles := make([]string, 0, len(files)+len(videoFiles))
	imageURLs := make([]string, 0, len(files))
	for _, file := range files {
		if file == nil {
			continue
		}
		filename := buildUploadFilename(file.Filename)
		dstPath := filepath.Join(s.uploadDir, filename)
		if err := c.SaveUploadedFile(file, dstPath); err != nil {
			s.cleanupPostUpload(postID, savedFiles)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
			return
		}

		publicURL := "/uploads/" + filename
		if err := s.addPostImage(postID, publicURL, now); err != nil {
			s.cleanupPostUpload(postID, append(savedFiles, dstPath))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
			return
		}
		savedFiles = append(savedFiles, dstPath)
		imageURLs = append(imageURLs, publicURL)
	}

	videoURLs := make([]string, 0, len(videoFiles))
	for _, file := range videoFiles {
		if file == nil {
			continue
		}
		filename := buildUploadFilename(file.Filename)
		dstPath := filepath.Join(s.uploadDir, filename)
		if err := c.SaveUploadedFile(file, dstPath); err != nil {
			s.cleanupPostUpload(postID, append(savedFiles, dstPath))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "视频保存失败"})
			return
		}

		publicURL := "/uploads/" + filename
		if err := s.addPostVideo(postID, publicURL, now); err != nil {
			s.cleanupPostUpload(postID, append(savedFiles, dstPath))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
			return
		}
		savedFiles = append(savedFiles, dstPath)
		videoURLs = append(videoURLs, publicURL)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "发布成功",
		"id":      postID,
		"images":  imageURLs,
		"videos":  videoURLs,
		"content": content,
		"tag_id":  tagID,
		"created": now,
	})
}

func (s *Server) handlePostList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		offset = parsed
	}

	posts, hasMore, err := s.listPosts(userIDStr, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	nextOffset := offset + len(posts)
	c.JSON(http.StatusOK, gin.H{
		"posts":       posts,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) handlePostRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	post, err := s.getPostByID(userIDStr, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if post == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到帖子"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"post": post})
}

func (s *Server) handlePostLike(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	if err := s.likePost(postID, userIDStr, time.Now()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已点赞"})
}

func (s *Server) handlePostUnlike(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	if err := s.unlikePost(postID, userIDStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "已取消点赞"})
}

func (s *Server) handleReplyCreate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "回复不能为空"})
		return
	}

	replyID, err := s.createReply(postID, userIDStr, content, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "回复成功",
		"id":      replyID,
	})
}

func (s *Server) handleReplyList(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || postID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子"})
		return
	}

	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		offset = parsed
	}

	replies, hasMore, err := s.listReplies(postID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	nextOffset := offset + len(replies)
	c.JSON(http.StatusOK, gin.H{
		"replies":     replies,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) cleanupPostUpload(postID int64, files []string) {
	_ = s.deletePost(postID)
	for _, path := range files {
		_ = os.Remove(path)
	}
}

func isUploadType(file *multipart.FileHeader, typePrefix string) bool {
	if file == nil {
		return false
	}
	contentType := strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type")))
	if contentType == "" {
		return false
	}
	return strings.HasPrefix(contentType, typePrefix)
}
