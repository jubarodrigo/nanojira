package middleware

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rodrigocavalhero/nanojira/internal/domain"
)

const UserIDHeader = "X-User-ID"

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader(UserIDHeader)
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing X-User-ID header (simulated auth)",
			})
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	return c.GetString("user_id")
}

func WriteError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "an unexpected error occurred",
		})
		return
	}

	status := http.StatusBadRequest
	switch {
	case errors.Is(appErr.Err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(appErr.Err, domain.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(appErr.Err, domain.ErrInvalidInput),
		errors.Is(appErr.Err, domain.ErrInvalidTransition),
		errors.Is(appErr.Err, domain.ErrPendingStepBack),
		errors.Is(appErr.Err, domain.ErrInvalidStepBack):
		status = http.StatusBadRequest
	case errors.Is(appErr.Err, domain.ErrEmailSend):
		status = http.StatusBadGateway
	}

	c.JSON(status, gin.H{
		"code":    appErr.Code,
		"message": appErr.Message,
	})
}
