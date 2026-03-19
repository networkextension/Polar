package dock

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		session := s.getSession(sessionID)
		if session == nil {
			c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)
		if session.Role == "" {
			c.Set("role", "user")
		} else {
			c.Set("role", session.Role)
		}
		c.Set("session", session)
		c.Next()
	}
}

func (s *Server) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if roleStr, ok := role.(string); !ok || roleStr != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) GuestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(SessionCookieName)
		if err == nil {
			session := s.getSession(sessionID)
			if session != nil {
				c.Redirect(http.StatusFound, "/dashboard")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func (s *Server) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=3"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	existingUser, err := s.getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
		return
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	user := &User{
		ID:        generateSessionID()[:16],
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		CreatedAt: time.Now(),
	}
	if err := s.createUser(user); err != nil {
		if err == errEmailExists {
			c.JSON(http.StatusConflict, gin.H{"error": "该邮箱已被注册"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	sessionID, err := s.createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "register")

	c.JSON(http.StatusCreated, gin.H{
		"message":  "注册成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	user, err := s.getUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if user == nil || !checkPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	sessionID, err := s.createSession(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.SetCookie(SessionCookieName, sessionID, int(SessionDuration.Seconds()), "/", "", false, true)
	s.recordLoginEvent(c, user.ID, "password")

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"user_id":  user.ID,
		"username": user.Username,
	})
}

func (s *Server) handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie(SessionCookieName)
	if err == nil {
		_ = s.deleteSession(sessionID)
	}

	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "已成功退出登录"})
}

func (s *Server) handleMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	})
}

func (s *Server) handleLoginHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
			return
		}
		limit = parsed
	}

	records, err := s.listLoginRecords(userIDStr, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"records": records,
	})
}

func (s *Server) handleMarkdownSubmit(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	if err := os.MkdirAll(s.markdownDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	safeTitle := sanitizeFilename(req.Title)
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	filename := safeTitle + "_" + timestamp + "_" + sanitizeFilename(fmt.Sprintf("%v", userID)) + ".md"
	path := filepath.Join(s.markdownDir, filename)

	content := req.Content
	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		content = "# " + req.Title + "\n\n" + content
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	entryID, err := s.createMarkdownEntryReturningID(userIDStr, req.Title, path, now)
	if err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "保存成功",
		"id":       entryID,
		"file":     path,
		"username": username,
	})
}

func (s *Server) handleMarkdownList(c *gin.Context) {
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

	entries, hasMore, err := s.listMarkdownEntries(userIDStr, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	nextOffset := offset + len(entries)
	c.JSON(http.StatusOK, gin.H{
		"entries":     entries,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) handleMarkdownRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	entryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	entry, err := s.getMarkdownEntry(userIDStr, entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到记录"})
		return
	}

	content, err := os.ReadFile(entry.FilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entry":   entry,
		"content": string(content),
	})
}

func (s *Server) handleMarkdownUpdate(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	entryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	entry, err := s.getMarkdownEntry(userIDStr, entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到记录"})
		return
	}

	content := req.Content
	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		content = "# " + req.Title + "\n\n" + content
	}

	if err := os.WriteFile(entry.FilePath, []byte(content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	if err := s.updateMarkdownEntry(userIDStr, entryID, req.Title, entry.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"id":      entryID,
	})
}

func (s *Server) handleMarkdownDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	entryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || entryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	entry, err := s.getMarkdownEntry(userIDStr, entryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到记录"})
		return
	}

	_ = os.Remove(entry.FilePath)
	if err := s.deleteMarkdownEntry(userIDStr, entryID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (s *Server) handleTagCreate(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	slug := normalizeTagSlug(req.Slug)
	if slug == "" {
		slug = normalizeTagSlug(req.Name)
	}
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的标签标识"})
		return
	}

	tag := &Tag{
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Description: strings.TrimSpace(req.Description),
		SortOrder:   req.SortOrder,
	}

	created, err := s.createTag(tag)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "标签标识已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"tag":     created,
	})
}

func (s *Server) handleTagList(c *gin.Context) {
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

	tags, hasMore, err := s.listTags(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	nextOffset := offset + len(tags)
	c.JSON(http.StatusOK, gin.H{
		"tags":        tags,
		"has_more":    hasMore,
		"next_offset": nextOffset,
	})
}

func (s *Server) handleTagUpdate(c *gin.Context) {
	tagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || tagID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	slug := normalizeTagSlug(req.Slug)
	if slug == "" {
		slug = normalizeTagSlug(req.Name)
	}
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的标签标识"})
		return
	}

	if err := s.updateTag(
		tagID,
		strings.TrimSpace(req.Name),
		slug,
		strings.TrimSpace(req.Description),
		req.SortOrder,
	); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "标签标识已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"id":      tagID,
	})
}

func (s *Server) handleTagDelete(c *gin.Context) {
	tagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || tagID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return
	}

	if err := s.deleteTag(tagID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

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
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要上传一张图片"})
		return
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

	savedFiles := make([]string, 0, len(files))
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

	if len(imageURLs) == 0 {
		s.cleanupPostUpload(postID, savedFiles)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "发布成功",
		"id":      postID,
		"images":  imageURLs,
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

func sanitizeFilename(input string) string {
	if input == "" {
		return "untitled"
	}
	var b strings.Builder
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "untitled"
	}
	return out
}

func normalizeTagSlug(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}
	return strings.ToLower(sanitizeFilename(input))
}

func buildUploadFilename(original string) string {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" || len(ext) > 8 {
		ext = ".img"
	} else {
		for _, r := range ext[1:] {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
				ext = ".img"
				break
			}
		}
	}
	return fmt.Sprintf("%s_%s%s", time.Now().Format("20060102_150405"), generateSessionID()[:8], ext)
}

func (s *Server) cleanupPostUpload(postID int64, files []string) {
	_ = s.deletePost(postID)
	for _, path := range files {
		_ = os.Remove(path)
	}
}
