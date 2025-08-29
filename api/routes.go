package api

import (
	"net/http"

	"github.com/blood-vessel/vitals/assert"
	"github.com/charmbracelet/log"

	"github.com/labstack/echo/v4"
)

func registerRoutes(
	e *echo.Echo,
	logger *log.Logger,
) {
	assert.AssertNotNil(e)
	assert.AssertNotNil(logger)

	e.GET("", func(c echo.Context) error {
		logger.Debug("vitals")
		return c.String(http.StatusOK, "blood-vessel/vitals")
	})
}
