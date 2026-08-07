package service

import (
	"context"
	"fmt"

	admindashboardmodel "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/model"
)

type Repository interface {
	Get(context.Context) (admindashboardmodel.Dashboard, error)
}

type Service struct {
	repository Repository
}

func New(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Get(ctx context.Context) (admindashboardmodel.Dashboard, error) {
	dashboard, err := s.repository.Get(ctx)
	if err != nil {
		return admindashboardmodel.Dashboard{}, fmt.Errorf("get dashboard statistics: %w", err)
	}

	return dashboard, nil
}
