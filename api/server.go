package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/blood-vessel/vitals/assert"
	"github.com/charmbracelet/log"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/workos/workos-go/v4/pkg/sso"
	"golang.org/x/time/rate"
)

type RunOptions struct {
	Writer   io.Writer
	Listener net.Listener
	Config   *viper.Viper
}

func Run(ctx context.Context, opts *RunOptions) error {
	assert.AssertNotNil(opts)
	assert.AssertNotNil(opts.Config)
	assert.AssertNotNil(opts.Listener)
	assert.AssertNotNil(opts.Writer)

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	logger := log.Default()
	logger.SetOutput(opts.Writer)
	if opts.Config.GetString("ENVIRONMENT") == "dev" {
		logger.SetLevel(log.DebugLevel)
	}

	deps := initDeps(opts.Config)
	assert.AssertNotNil(deps)

	server := newServer(
		logger,
		opts.Config,
		deps,
	)
	assert.AssertNotNil(server)

	go func() {
		log.Info("Serving")
		err := server.Serve(opts.Listener)
		if err != nil && err != http.ErrServerClosed {
			logger.Fatal("Received error from http server", "err", err)
		}
	}()

	<-ctx.Done()

	timeout := time.Second * 10
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("err during shutdown %w", err)
	}

	return nil
}

type ServerDependencies struct {
	SSOCient *sso.Client
}

func initDeps(config *viper.Viper) *ServerDependencies {
	assert.AssertNotNil(config)

	ssoClient := &sso.Client{
		APIKey:   config.GetString("WORKOS_API_KEY"),
		ClientID: config.GetString("WORKOS_CLIENT_ID"),
	}

	assert.AssertNotNil(ssoClient.APIKey)
	assert.AssertNotNil(ssoClient.ClientID)
	return &ServerDependencies{
		SSOCient: ssoClient,
	}
}

func newServer(logger *log.Logger, config *viper.Viper, deps *ServerDependencies) *http.Server {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(config)
	assert.AssertNotNil(deps)

	e := echo.New()
	e.IPExtractor = echo.ExtractIPDirect()

	server := &http.Server{
		Handler:           e,
		ReadTimeout:       5 * time.Minute,
		IdleTimeout:       30 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          logger.StandardLog(),
	}

	eLoggerConfig := middleware.DefaultLoggerConfig
	eLoggerConfig.Output = logger.StandardLog().Writer()
	e.Use(middleware.LoggerWithConfig(eLoggerConfig))

	rlimiterConfig := rateLimiterConfig(logger)
	e.Use(middleware.RateLimiterWithConfig(rlimiterConfig))

	corsConfig := corsConfig()
	e.Use(middleware.CORSWithConfig(corsConfig))

	e.Validator = &CustomValidator{validator: validator.New()}

	registerRoutes(e, logger, config, deps)

	assert.AssertNotNil(server)
	return server
}

func rateLimiterConfig(logger *log.Logger) middleware.RateLimiterConfig {
	config := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(5),
				Burst:     15,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		DenyHandler: func(ctx echo.Context, identifier string, err error) error {
			logger.Warn("ratelimiting", "identifier", identifier, "err", err)
			return ctx.NoContent(http.StatusTooManyRequests)
		},
	}

	return config
}

func corsConfig() middleware.CORSConfig {
	config := middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderXRequestedWith,
			echo.HeaderAuthorization,
		},
	}

	return config
}

type CustomValidator struct {
	validator *validator.Validate
}

func (v *CustomValidator) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return nil
}
