package server

import (
	"net/http"

	"github.com/lemonc7/silo/service"
	"github.com/lemonc7/zest"
)

type Server struct {
	service *service.Service
}

func New(s *service.Service) *Server {
	return &Server{
		service: s,
	}
}

func (s *Server) GetMedias(c *zest.Context) error {
	medias, err := s.service.GetMedias(c.Context())
	if err != nil {
		return zest.NewHTTPError(http.StatusInternalServerError, "获取媒体信息失败").Wrap(err)
	}

	return c.JSON(http.StatusOK, medias)
}

func (s *Server) GetSeasons(c *zest.Context) error {
	var req GetSeasonsRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	seasons, err := s.service.GetSeasons(c.Context(), req.SeasonID)
	if err != nil {
		return zest.NewHTTPError(http.StatusInternalServerError, "获取季信息").Wrap(err)
	}

	return c.JSON(http.StatusOK, seasons)
}

func (s *Server) GetEpisodes(c *zest.Context) error {
	var req GetEpisodesRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	episodes, err := s.service.GetEpisodes(c.Context(), req.EpisodeID)
	if err != nil {
		return zest.NewHTTPError(http.StatusInternalServerError, "获取集信息").Wrap(err)
	}

	return c.JSON(http.StatusOK, episodes)
}
