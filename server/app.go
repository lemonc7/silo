package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lemonc7/silo/config"
	"github.com/lemonc7/silo/service"
	"github.com/lemonc7/zest"
	"github.com/lemonc7/zest/middleware"
)

type App struct {
	zest   *zest.Zest
	server *http.Server
	cfg    config.Config
}

func InitApp(cfg config.Config, service *service.Service) *App {
	z := zest.New()
	z.StructValidator = &Validator{Validate: validator.New()}
	z.Use(middleware.RequestID())
	z.Use(middleware.Logger(middleware.LoggerConfig{
		TZ: cfg.Log.TZ,
	}))
	z.Use(middleware.Recovery())
	z.Use(middleware.CORS())

	z.GET("/health", func(c *zest.Context) error {
		return c.String(http.StatusOK, "Welcome to Silo!")
	})

	handler := New(service)
	api := z.Group("/api")

	api.GET("/media", handler.GetMedias)
	api.GET("/media/{seasonID}", handler.GetSeasons)
	api.GET("/media/{seasonID}/{episodeID}", handler.GetEpisodes)

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Server.Port),
		Handler:      z,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		zest:   z,
		server: srv,
		cfg:    cfg,
	}
}

func (a *App) Run(ctx context.Context) {
	slog.Info("HTTP服务已启动",
		slog.String("component", "http server"),
		slog.Int("port", a.cfg.Server.Port),
	)
	go func() {
		if err := a.server.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				slog.Error("启动HTTP服务失败",
					slog.String("component", "http server"),
					slog.String("err", err.Error()),
				)
			}
		}
	}()

	<-ctx.Done()

	slog.Info("收到退出信号, 正在关闭HTTP服务...",
		slog.String("component", "http server"))

	ctx, stop := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer stop()

	if err := a.server.Shutdown(ctx); err != nil {
		slog.Error("关闭HTTP服务失败",
			slog.String("component", "http server"),
			slog.String("err", err.Error()),
		)
	} else {
		slog.Info("成功关闭HTTP服务", slog.String("component", "http server"))
	}
}
