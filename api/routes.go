package api

import (
	"net/http"

	"github.com/blood-vessel/vitals/assert"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"

	"github.com/labstack/echo/v4"
)

func registerRoutes(
	e *echo.Echo,
	logger *log.Logger,
	config *viper.Viper,
	deps *ServerDependencies,
) {
	assert.AssertNotNil(e)
	assert.AssertNotNil(logger)
	assert.AssertNotNil(deps)
	assert.AssertNotNil(config)

	e.GET("", func(c echo.Context) error {
		logger.Debug("vitals")
		return c.String(http.StatusOK, "blood-vessel/vitals")
	})
}

func bindAndValidate[T any](c echo.Context) (*T, error) {
	var req T
	err := c.Bind(&req)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	err = c.Validate(req)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return &req, nil
}
