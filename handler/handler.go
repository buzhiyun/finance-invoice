package handler

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/buzhiyun/finance-invoice/auth"
	"github.com/buzhiyun/finance-invoice/task"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	TaskManager *task.Manager
	UserStore   *auth.UserStore
	JWTSecret   string
}

func New(tm *task.Manager, us *auth.UserStore, jwtSecret string) *Handler {
	return &Handler{
		TaskManager: tm,
		UserStore:   us,
		JWTSecret:   jwtSecret,
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	if !h.UserStore.Authenticate(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, err := auth.GenerateToken(req.Username, h.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解析上传表单"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择至少一个PDF文件"})
		return
	}

	fileData := make(map[string][]byte)
	for _, fh := range files {
		if filepath.Ext(fh.Filename) != ".pdf" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持PDF文件: " + fh.Filename})
			return
		}

		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件: " + fh.Filename})
			return
		}

		data := make([]byte, fh.Size)
		if _, err := f.Read(data); err != nil {
			f.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件: " + fh.Filename})
			return
		}
		f.Close()

		fileData[fh.Filename] = data
	}

	t := h.TaskManager.CreateTask(fileData)
	c.JSON(http.StatusOK, gin.H{
		"task_id": t.ID,
		"status":  t.Status,
		"message": fmt.Sprintf("已创建任务，共 %d 个文件", len(fileData)),
	})
}

func (h *Handler) ListTasks(c *gin.Context) {
	tasks := h.TaskManager.ListTasks()
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (h *Handler) ClearTasks(c *gin.Context) {
	removed := h.TaskManager.ClearFinished()
	c.JSON(http.StatusOK, gin.H{"removed": removed})
}

func (h *Handler) GetTask(c *gin.Context) {
	id := c.Param("id")
	t := h.TaskManager.GetTask(id)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func (h *Handler) DownloadExcel(c *gin.Context) {
	id := c.Param("id")
	t := h.TaskManager.GetTask(id)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if t.ExcelPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Excel文件尚未生成"})
		return
	}

	c.FileAttachment(t.ExcelPath, fmt.Sprintf("识别结果_%s.xlsx", id))
}
