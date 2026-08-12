package kwork

import (
	"context"

	"kwork-assistant/internal/domain"
)

// KworkProjectSource implements the domain.ProjectSource interface.
type KworkProjectSource struct {
	client    *Client
	token     string
	login     string
	password  string
	phoneLast string
}

// NewKworkProjectSource creates a new source adapter.
func NewKworkProjectSource(login, password, phoneLast string) *KworkProjectSource {
	return &KworkProjectSource{
		client:    NewClient(),
		login:     login,
		password:  password,
		phoneLast: phoneLast,
	}
}

// ensureAuth checks if token exists, and authenticates if not.
func (s *KworkProjectSource) ensureAuth(ctx context.Context) error {
	if s.token != "" {
		return nil
	}
	token, err := s.client.Authenticate(ctx, s.login, s.password, s.phoneLast)
	if err != nil {
		return err
	}
	s.token = token
	return nil
}

// FetchCategories fetches categories using the unified client.
func (s *KworkProjectSource) FetchCategories(ctx context.Context) (map[int]string, error) {
	if err := s.ensureAuth(ctx); err != nil {
		return nil, err
	}
	return s.client.FetchCategories(ctx, s.token)
}

// FetchProjects fetches projects using the unified client.
func (s *KworkProjectSource) FetchProjects(ctx context.Context, limit int) ([]domain.Project, error) {
	if err := s.ensureAuth(ctx); err != nil {
		return nil, err
	}
	return s.client.FetchProjects(ctx, s.token, limit)
}

// Health checks if authentication works.
func (s *KworkProjectSource) Health(ctx context.Context) error {
	return s.ensureAuth(ctx)
}

// GetMe fetches user data and returns ID and Username.
func (s *KworkProjectSource) GetMe(ctx context.Context) (int64, string, error) {
	if err := s.ensureAuth(ctx); err != nil {
		return 0, "", err
	}
	actor, err := s.client.GetMe(ctx, s.token)
	if err != nil {
		return 0, "", err
	}
	return int64(actor.ID), actor.Username, nil
}
