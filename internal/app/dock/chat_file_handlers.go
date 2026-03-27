package dock

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// handleChatFileProxy serves a chat attachment that is stored in remote object
// storage (Cloudflare R2) by fetching it via the authenticated S3 API and
// streaming the bytes back to the client.
//
// Route: GET /api/chat-files/:filename  (requires authentication)
//
// This endpoint is only reachable for R2-stored files whose URL was generated
// in "proxy mode" (i.e. CF_R2_PUBLIC_URL was not set at startup).  Files
// stored on the local filesystem are served by the /uploads/ static handler
// and never reach this handler.
func (s *Server) handleChatFileProxy(c *gin.Context) {
	filename := c.Param("filename")

	// Basic safety: reject anything with path separators or traversal sequences.
	if strings.ContainsAny(filename, "/\\") || strings.Contains(filename, "..") || filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	body, size, contentType, err := s.chatStorage.GetObject(c.Request.Context(), filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	defer body.Close()

	// Fall back to MIME detection from extension when the store doesn't provide it.
	if contentType == "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if t := mime.TypeByExtension(ext); t != "" {
			contentType = t
		} else {
			contentType = "application/octet-stream"
		}
	}

	c.Header("Content-Type", contentType)
	if size > 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	// Allow clients and CDN edge caches to cache the file for 24 h.
	// "private" ensures shared caches (proxies) do not store user files.
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")

	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, body)
}
