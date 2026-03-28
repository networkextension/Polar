package dock

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type packTunnelProfileRequest struct {
	ID        string                    `json:"id"`
	Name      string                    `json:"name" binding:"required"`
	Type      string                    `json:"type" binding:"required"`
	Server    PackTunnelServerEndpoint  `json:"server"`
	Auth      PackTunnelAuth            `json:"auth"`
	Options   PackTunnelOptions         `json:"options"`
	Transport *PackTunnelTransport      `json:"transport,omitempty"`
	Metadata  packTunnelMetadataRequest `json:"metadata"`
}

type packTunnelMetadataRequest struct {
	Priority    int    `json:"priority"`
	Enabled     *bool  `json:"enabled"`
	Editable    *bool  `json:"editable"`
	Source      string `json:"source"`
	CountryCode string `json:"country_code"`
	CountryFlag string `json:"country_flag"`
	IsActive    bool   `json:"is_active"`
}

func (s *Server) handlePackTunnelProfileList(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel list ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	items, err := s.listPackTunnelProfiles(systemUserID)
	if err != nil {
		log.Printf("packtunnel list profiles failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	active, err := s.getActivePackTunnelProfile(systemUserID)
	if err != nil {
		log.Printf("packtunnel get active profile failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"profiles":       items,
		"active_profile": active,
	})
}

func (s *Server) handlePackTunnelProfileGet(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel get ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}
	item, err := s.getPackTunnelProfile(systemUserID, id)
	if err != nil {
		log.Printf("packtunnel get profile failed id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": item})
}

func (s *Server) handlePackTunnelActiveProfileGet(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel active ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	item, err := s.getActivePackTunnelProfile(systemUserID)
	if err != nil {
		log.Printf("packtunnel active get failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "当前没有启用配置"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": item})
}

func (s *Server) handlePackTunnelProfileCreate(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel create ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	item, ok := buildPackTunnelProfileFromRequest(c)
	if !ok {
		return
	}

	created, err := s.createPackTunnelProfile(systemUserID, item, time.Now())
	if err != nil {
		log.Printf("packtunnel create failed name=%s type=%s: %v", item.Name, item.Type, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"profile": created, "message": "配置已创建"})
}

func (s *Server) handlePackTunnelProfileUpdate(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel update ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}

	item, ok := buildPackTunnelProfileFromRequest(c)
	if !ok {
		return
	}
	item.ID = id

	updated, err := s.updatePackTunnelProfile(systemUserID, id, item, time.Now())
	if err != nil {
		log.Printf("packtunnel update failed id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": updated, "message": "配置已更新"})
}

func (s *Server) handlePackTunnelProfileDelete(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel delete ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}
	ok, err := s.deletePackTunnelProfile(systemUserID, id)
	if err != nil {
		log.Printf("packtunnel delete failed id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "配置已删除"})
}

func (s *Server) handlePackTunnelProfileActivate(c *gin.Context) {
	if err := s.ensureSystemUser(); err != nil {
		log.Printf("packtunnel activate ensure system user failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务器错误"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效配置"})
		return
	}

	item, err := s.setActivePackTunnelProfile(systemUserID, id, time.Now())
	if err != nil {
		log.Printf("packtunnel activate failed id=%s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启用失败"})
		return
	}
	if item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "配置不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": item, "message": "配置已启用"})
}

func buildPackTunnelProfileFromRequest(c *gin.Context) (PackTunnelProfile, bool) {
	var req packTunnelProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的输入数据"})
		return PackTunnelProfile{}, false
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	req.Server.Address = strings.TrimSpace(req.Server.Address)
	req.Auth.Password = strings.TrimSpace(req.Auth.Password)
	req.Auth.Method = strings.TrimSpace(req.Auth.Method)
	req.Metadata.Source = strings.TrimSpace(req.Metadata.Source)
	req.Metadata.CountryCode = strings.TrimSpace(req.Metadata.CountryCode)
	req.Metadata.CountryFlag = strings.TrimSpace(req.Metadata.CountryFlag)

	if req.Name == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称和类型不能为空"})
		return PackTunnelProfile{}, false
	}
	if req.Server.Address == "" || req.Server.Port <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "服务器地址和端口不能为空"})
		return PackTunnelProfile{}, false
	}
	if req.Metadata.Source == "" {
		req.Metadata.Source = "local"
	}
	enabled := true
	if req.Metadata.Enabled != nil {
		enabled = *req.Metadata.Enabled
	}
	editable := true
	if req.Metadata.Editable != nil {
		editable = *req.Metadata.Editable
	}
	if req.Transport != nil {
		req.Transport.Kind = strings.TrimSpace(req.Transport.Kind)
		if req.Transport.Kind == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "传输层类型不能为空"})
			return PackTunnelProfile{}, false
		}
		if req.Transport.Kind == "kcptun" && req.Transport.KCPTun == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kcptun 配置不能为空"})
			return PackTunnelProfile{}, false
		}
	}

	return PackTunnelProfile{
		ID:   strings.TrimSpace(req.ID),
		Name: req.Name,
		Type: req.Type,
		Server: PackTunnelServerEndpoint{
			Address: req.Server.Address,
			Port:    req.Server.Port,
		},
		Auth: PackTunnelAuth{
			Password: req.Auth.Password,
			Method:   req.Auth.Method,
		},
		Options:   req.Options,
		Transport: req.Transport,
		Metadata: PackTunnelProfileMetadata{
			Priority:    req.Metadata.Priority,
			Enabled:     enabled,
			Editable:    editable,
			Source:      req.Metadata.Source,
			CountryCode: req.Metadata.CountryCode,
			CountryFlag: req.Metadata.CountryFlag,
			IsActive:    req.Metadata.IsActive,
		},
	}, true
}
