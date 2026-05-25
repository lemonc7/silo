package server

import (
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	Validate *validator.Validate
}

func (v *Validator) ValidateStruct(ptr any) error {
	return v.Validate.Struct(ptr)
}

type GetSeasonsRequest struct {
	SeasonID int64 `path:"seasonID" validate:"required"`
}

type GetEpisodesRequest struct {
	EpisodeID int64 `path:"episodeID" validate:"required"`
}
