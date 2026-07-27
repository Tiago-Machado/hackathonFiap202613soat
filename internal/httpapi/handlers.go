package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"video-processor/internal/usecase"
)

const (
	contextUserID  = "userID"
	formFileField  = "video"
	defaultPage    = "1"
	pageQueryParam = "page"
)

type Handlers struct {
	register *usecase.Register
	login    *usecase.Login
	upload   *usecase.UploadVideo
	list     *usecase.ListVideos
}

func NewHandlers(register *usecase.Register, login *usecase.Login, upload *usecase.UploadVideo, list *usecase.ListVideos) *Handlers {
	return &Handlers{register: register, login: login, upload: upload, list: list}
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handlers) Register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "e-mail e senha são obrigatórios"})
		return
	}
	user, err := h.register.Execute(c.Request.Context(), usecase.Credentials(req))
	if err != nil {
		c.JSON(registerErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "email": user.Email})
}

func (h *Handlers) Login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "e-mail e senha são obrigatórios"})
		return
	}
	token, err := h.login.Execute(c.Request.Context(), usecase.Credentials(req))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handlers) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile(formFileField)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "arquivo de vídeo ausente (campo 'video')"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "não foi possível ler o arquivo enviado"})
		return
	}
	defer file.Close()

	video, err := h.upload.Execute(c.Request.Context(), usecase.UploadVideoInput{
		UserID:      c.GetString(contextUserID),
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
		Content:     file,
	})
	if err != nil {
		c.JSON(uploadErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"id": video.ID, "status": video.Status})
}

func (h *Handlers) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery(pageQueryParam, defaultPage))
	views, err := h.list.Execute(c.Request.Context(), c.GetString(contextUserID), page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "não foi possível listar os vídeos"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"videos": toResponse(views)})
}

func toResponse(views []usecase.VideoView) []gin.H {
	items := make([]gin.H, 0, len(views))
	for _, view := range views {
		items = append(items, gin.H{
			"id":            view.Video.ID,
			"filename":      view.Video.OriginalFilename,
			"status":        view.Video.Status,
			"frame_count":   view.Video.FrameCount,
			"error_message": view.Video.ErrorMessage,
			"created_at":    view.Video.CreatedAt,
			"download_url":  view.DownloadURL,
		})
	}
	return items
}

func registerErrorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrEmailInUse):
		return http.StatusConflict
	case errors.Is(err, usecase.ErrWeakPassword), errors.Is(err, usecase.ErrInvalidEmail):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func uploadErrorStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrUnsupportedFormat):
		return http.StatusBadRequest
	case errors.Is(err, usecase.ErrQuotaExceeded):
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
