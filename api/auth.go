package api

import (
	"context"
	"crypto/rand"
	"net/http"
	"time"

	"github.com/blood-vessel/vitals/assert"
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"github.com/workos/workos-go/v4/pkg/sso"
)

const oauthStateCookieName = "oauth_state"

func handleLoginRedirect(logger *log.Logger, ssoClient *sso.Client, config *viper.Viper) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(ssoClient)
	assert.AssertNotNil(config)

	authCallbackURI := config.GetString("WORKOS_AUTH_CALLBACK")
	assert.AssertNotEmpty(authCallbackURI)
	return func(c echo.Context) error {
		state := rand.Text()
		url, err := ssoClient.GetAuthorizationURL(sso.GetAuthorizationURLOpts{
			RedirectURI: authCallbackURI,
			Provider:    sso.GitHubOAuth,
			State:       state,
		})
		if err != nil {
			logger.Error("sso get auth url", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		c.SetCookie(&http.Cookie{
			Name:     oauthStateCookieName,
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300, // 5 minutes
		})

		return c.Redirect(http.StatusFound, url.String())
	}
}

func handleAuthCallback(logger *log.Logger, ssoClient *sso.Client) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(ssoClient)
	type request struct {
		Code  string `query:"code" validate:"max=1024"`
		State string `query:"state" validate:"required,max=256"`
		Error string `query:"error_description" validate:"max=2048"`
	}
	return func(c echo.Context) error {
		req, err := bindAndValidate[request](c)
		if err != nil {
			return err
		}

		if req.Error != "" {
			logger.Warn("oauth error [error_description]", "err", req.Error)
			return c.NoContent(http.StatusBadRequest)
		}

		if req.Code == "" {
			logger.Warn("oauth error: no code returned from auth")
			return c.NoContent(http.StatusBadRequest)
		}

		st, err := c.Cookie(oauthStateCookieName)
		if err != nil {
			logger.Warn("err returning oauth state cookie", "err", err)
			return c.NoContent(http.StatusBadRequest)
		}
		if st == nil {
			logger.Warn("invalid oauth state: no cookie found")
			return c.NoContent(http.StatusBadRequest)
		}
		if st.Value == "" || st.Value != req.State {
			logger.Warn("invalid oauth state: value does not match")
			return c.NoContent(http.StatusBadRequest)
		}

		c.SetCookie(&http.Cookie{
			Name:     oauthStateCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
		defer cancel()
		opts := sso.GetProfileAndTokenOpts{Code: req.Code}
		_, err = ssoClient.GetProfileAndToken(ctx, opts)
		if err != nil {
			logger.Error("get profile", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.Redirect(http.StatusFound, "/")
	}
}
