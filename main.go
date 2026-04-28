package main

import (
	"fmt"
	"log"

	"github.com/buzhiyun/finance-invoice/auth"
	"github.com/buzhiyun/finance-invoice/config"
	"github.com/buzhiyun/finance-invoice/excel"
	"github.com/buzhiyun/finance-invoice/handler"
	"github.com/buzhiyun/finance-invoice/middleware"
	"github.com/buzhiyun/finance-invoice/task"
	"github.com/buzhiyun/finance-invoice/zhipu"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	userStore, err := auth.LoadUsers(cfg.UsersCSV)
	if err != nil {
		log.Fatalf("load users: %v", err)
	}

	zhipuClient := zhipu.NewClient(cfg.ZhipuAPIKey)

	taskManager, err := task.NewManager(zhipuClient, &excel.Generator{}, cfg.MaxConcurrent)
	if err != nil {
		log.Fatalf("create task manager: %v", err)
	}

	h := handler.New(taskManager, userStore, cfg.JWTSecret)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(middleware.IPWhitelist(cfg.AllowedIPs))

	r.StaticFile("/", "./web/index.html")

	r.POST("/api/login", h.Login)

	api := r.Group("/api")
	api.Use(middleware.Auth(cfg.JWTSecret))
	{
		api.POST("/upload", h.Upload)
		api.POST("/tasks/clear", h.ClearTasks)
		api.GET("/tasks", h.ListTasks)
		api.GET("/tasks/:id", h.GetTask)
		api.GET("/tasks/:id/download", h.DownloadExcel)
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s, max concurrent: %d", addr, cfg.MaxConcurrent)
	if len(cfg.AllowedIPs) > 0 {
		log.Printf("IP whitelist enabled: %d CIDR rules", len(cfg.AllowedIPs))
	}
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
