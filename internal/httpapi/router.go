package httpapi

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/Q1YZn/appleshare-hub/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.Service
	webFS   fs.FS
}

func NewRouter(svc *service.Service, webFS fs.FS) *gin.Engine {
	handler := &Handler{service: svc, webFS: webFS}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/healthz", handler.healthz)
	router.GET("/api/accounts", handler.accounts)
	router.GET("/api/status", handler.accounts)

	static, err := fs.Sub(webFS, "web")
	if err == nil {
		router.GET("/assets/*filepath", func(c *gin.Context) {
			file := path.Join("assets", strings.TrimPrefix(c.Param("filepath"), "/"))
			data, readErr := fs.ReadFile(static, file)
			if readErr != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, mime.TypeByExtension(path.Ext(file)), data)
		})
	}
	router.GET("/", handler.index)

	return router
}

func (h *Handler) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

func (h *Handler) accounts(c *gin.Context) {
	snapshot, _ := h.service.Snapshot(c.Request.Context())
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, snapshot)
}

func (h *Handler) index(c *gin.Context) {
	data, err := fs.ReadFile(h.webFS, "web/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "index.html not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
