package api

import (
	"net/http"

	"github.com/blood-vessel/vitals/assert"
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/workos/workos-go/v4/pkg/sso"
)

func handleLoginRedirect(logger *log.Logger, ssoClient *sso.Client, config *viper.Viper) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(ssoClient)
	assert.AssertNotNil(config)

	authCallbackURI := config.GetString("WORKOS_AUTH_CALLBACK")
	assert.AssertNotEmpty(authCallbackURI)
	return func(c echo.Context) error {
		url, err := ssoClient.GetAuthorizationURL(sso.GetAuthorizationURLOpts{
			RedirectURI: authCallbackURI,
			Provider:    sso.GitHubOAuth,
		})
		if err != nil {
			logger.Error("sso get auth url", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.Redirect(http.StatusFound, url.String())
	}
}

func handleAuthCallback(logger *log.Logger, ssoClient *sso.Client) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(ssoClient)
	type request struct {
		Code  string `query:"code" validate:"max=30"`
		Error string `query:"error_description" validate:"max=255"`
	}
	return func(c echo.Context) error {
		req, err := bindAndValidate[request](c)
		if err != nil {
			return err
		}

		if req.Error != "" {
			logger.Error(req.Error)
			return c.NoContent(http.StatusInternalServerError)
		}

		if req.Code == "" {
			logger.Error("no code returned from auth")
			return c.NoContent(http.StatusInternalServerError)
		}

		opts := sso.GetProfileAndTokenOpts{Code: req.Code}
		_, err = ssoClient.GetProfileAndToken(c.Request().Context(), opts)
		if err != nil {
			logger.Error("get profile", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.Redirect(http.StatusFound, "/")
	}
}
