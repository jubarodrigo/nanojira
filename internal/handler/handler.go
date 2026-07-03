package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
	"github.com/rodrigocavalhero/nanojira/internal/handler/middleware"
	"github.com/rodrigocavalhero/nanojira/internal/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", h.health)

	api := r.Group("/api/v1")
	api.Use(middleware.Auth())
	{
		api.GET("/tasks", h.listTasks)
		api.POST("/tasks", h.createTask)
		api.GET("/tasks/:id", h.getTask)
		api.PATCH("/tasks/:id/assign", h.assignTask)
		api.PATCH("/tasks/:id/status", h.updateStatus)
		api.GET("/tasks/:id/notifications", h.listNotifications)
	}
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type createTaskRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description"`
	AssigneeID  *string `json:"assignee_id"`
}

func (h *Handler) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.WriteError(c, domain.InvalidInput(err.Error()))
		return
	}

	task, err := h.svc.CreateTask(c.Request.Context(), middleware.GetUserID(c), service.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) listTasks(c *gin.Context) {
	var status *domain.TaskStatus
	if s := c.Query("status"); s != "" {
		st := domain.TaskStatus(s)
		if !st.Valid() {
			middleware.WriteError(c, domain.InvalidInput("invalid status filter"))
			return
		}
		status = &st
	}

	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)

	result, err := h.svc.ListTasks(c.Request.Context(), middleware.GetUserID(c), service.ListTasksInput{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) getTask(c *gin.Context) {
	task, err := h.svc.GetTask(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

type assignTaskRequest struct {
	AssigneeID string `json:"assignee_id" binding:"required"`
}

func (h *Handler) assignTask(c *gin.Context) {
	var req assignTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.WriteError(c, domain.InvalidInput(err.Error()))
		return
	}

	task, err := h.svc.AssignTask(c.Request.Context(), middleware.GetUserID(c), c.Param("id"), service.AssignTaskInput{
		AssigneeID: req.AssigneeID,
	})
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

type updateStatusRequest struct {
	Status              string `json:"status"`
	Reason              string `json:"reason"`
	ApproveStatusChange *bool  `json:"approve_status_change"`
}

func (h *Handler) updateStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.WriteError(c, domain.InvalidInput(err.Error()))
		return
	}

	input := service.UpdateStatusInput{
		Reason:              req.Reason,
		ApproveStatusChange: req.ApproveStatusChange,
	}

	if req.ApproveStatusChange == nil {
		if req.Status == "" {
			middleware.WriteError(c, domain.InvalidInput("status is required"))
			return
		}
		status := domain.TaskStatus(req.Status)
		if !status.Valid() {
			middleware.WriteError(c, domain.InvalidInput("invalid status value"))
			return
		}
		input.Status = status
	}

	task, err := h.svc.UpdateTaskStatus(c.Request.Context(), middleware.GetUserID(c), c.Param("id"), input)
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) listNotifications(c *gin.Context) {
	notifications, err := h.svc.GetAssignmentNotifications(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		middleware.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifications})
}

func parseIntDefault(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return fallback
	}
	return n
}
