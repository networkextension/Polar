package dock

import (
	"context"
	"log"
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

	postType := strings.TrimSpace(c.PostForm("post_type"))
	if postType == "" {
		postType = "standard"
	}
	if postType != "standard" && postType != "task" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子类型"})
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

	var (
		taskLocation      string
		taskStartAt       time.Time
		taskEndAt         time.Time
		taskApplyDeadline time.Time
		taskWorkingHours  string
	)
	if postType == "task" {
		taskLocation = strings.TrimSpace(c.PostForm("task_location"))
		taskWorkingHours = strings.TrimSpace(c.PostForm("working_hours"))
		if taskWorkingHours == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务帖必须填写 working hours"})
			return
		}
		startAtStr := strings.TrimSpace(c.PostForm("task_start_at"))
		endAtStr := strings.TrimSpace(c.PostForm("task_end_at"))
		deadlineStr := strings.TrimSpace(c.PostForm("apply_deadline"))
		if startAtStr == "" || endAtStr == "" || deadlineStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务帖必须填写时间范围和申请截止时间"})
			return
		}
		var err error
		taskStartAt, err = time.Parse(time.RFC3339, startAtStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务开始时间格式错误"})
			return
		}
		taskEndAt, err = time.Parse(time.RFC3339, endAtStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务结束时间格式错误"})
			return
		}
		taskApplyDeadline, err = time.Parse(time.RFC3339, deadlineStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "申请截止时间格式错误"})
			return
		}
		if !taskStartAt.Before(taskEndAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "任务开始时间必须早于结束时间"})
			return
		}
		if !taskApplyDeadline.Before(taskStartAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "申请截止时间必须早于任务开始时间"})
			return
		}
	}

	now := time.Now()
	postID, err := s.createPost(userIDStr, tagID, postType, content, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if postType == "task" {
		if err := s.createTaskPost(postID, taskLocation, taskStartAt, taskEndAt, taskWorkingHours, taskApplyDeadline); err != nil {
			_, _ = s.deletePost(postID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "任务创建失败"})
			return
		}
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
	videoItems := make([]PostVideo, 0, len(videoFiles))
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
		posterURL := ""
		posterFilename := buildDerivedUploadFilename(filename, "poster", ".jpg")
		posterPath := filepath.Join(s.uploadDir, posterFilename)
		posterPublicURL := "/uploads/" + posterFilename

		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		err := generateVideoPoster(ctx, dstPath, posterPath)
		cancel()
		if err != nil {
			log.Printf("generate video poster failed for %s: %v", dstPath, err)
		} else {
			savedFiles = append(savedFiles, posterPath)
			posterURL = posterPublicURL
		}

		if err := s.addPostVideo(postID, publicURL, posterURL, now); err != nil {
			s.cleanupPostUpload(postID, append(savedFiles, dstPath))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
			return
		}
		savedFiles = append(savedFiles, dstPath)
		videoURLs = append(videoURLs, publicURL)
		videoItems = append(videoItems, PostVideo{URL: publicURL, PosterURL: posterURL})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "发布成功",
		"id":          postID,
		"post_type":   postType,
		"images":      imageURLs,
		"videos":      videoURLs,
		"video_items": videoItems,
		"content":     content,
		"tag_id":      tagID,
		"created":     now,
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

	var tagID *int64
	if tagIDStr := strings.TrimSpace(c.Query("tag_id")); tagIDStr != "" {
		parsed, err := strconv.ParseInt(tagIDStr, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的标签"})
			return
		}
		tagID = &parsed
	}

	postType := strings.TrimSpace(c.Query("post_type"))
	if postType == "" {
		postType = "all"
	}
	if postType != "all" && postType != "standard" && postType != "task" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的帖子类型"})
		return
	}

	posts, hasMore, err := s.listPosts(userIDStr, limit, offset, tagID, postType)
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

func (s *Server) handlePostDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	role, _ := c.Get("role")
	roleStr, _ := role.(string)

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
	if roleStr != "admin" && post.UserID != userIDStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	deleted, err := s.deletePost(postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if !deleted {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到帖子"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "帖子已删除"})
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
	_, _ = s.deletePost(postID)
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
