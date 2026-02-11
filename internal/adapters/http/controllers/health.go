package controllers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthResponse struct {
	Status   string            `json:"status" example:"ok"`
	Services map[string]string `json:"services" example:"mongodb:ok,redis:ok,rabbitmq:ok"`
}

type HealthChecker struct {
	Name  string
	Check func(ctx context.Context) error
}

type HealthController struct {
	checkers []HealthChecker
}

func NewHealthController(checkers []HealthChecker) *HealthController {
	return &HealthController{checkers: checkers}
}

// Health godoc
// @Summary     Health check
// @Description Checks the health of all dependent services
// @Tags        health
// @Produce     json
// @Success     200 {object} HealthResponse
// @Failure     503 {object} HealthResponse
// @Router      /api/v1/health [get]
func (h *HealthController) Health(c *gin.Context) {
	ctx := c.Request.Context()

	status := "ok"
	services := make(map[string]string, len(h.checkers))

	for _, checker := range h.checkers {
		if err := checker.Check(ctx); err != nil {
			services[checker.Name] = err.Error()
			status = "degraded"
		} else {
			services[checker.Name] = "ok"
		}
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, HealthResponse{
		Status:   status,
		Services: services,
	})
}
