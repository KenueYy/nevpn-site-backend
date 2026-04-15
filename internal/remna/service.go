package remna

import (
	"context"
	"log/slog"
)

type IRemnaService interface {
	GetAllUsers(ctx context.Context) (*AllUserResponse, error)
	GetUserByUUID(ctx context.Context, id string) (*RemnaUserResponse, error)
	GetUserByEmail(ctx context.Context, email string) (*RemnaUserResponse, error)
	GetUserByTelegramID(ctx context.Context, tgID string) (*RemnaUserResponse, error)
	CreateNewUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error)
	UpdateUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error)
}

type RemnaService struct {
	client IRemnaClient
	logger *slog.Logger
}

func NewService(remnaClient IRemnaClient, logger *slog.Logger) *RemnaService {
	return &RemnaService{
		client: remnaClient,
		logger: logger,
	}
}

func (s *RemnaService) GetAllUsers(ctx context.Context) (*AllUserResponse, error) {
	return s.client.GetAllUser(ctx)
}

func (s *RemnaService) GetUserByUUID(ctx context.Context, id string) (*RemnaUserResponse, error) {
	return s.client.GetUserByUUID(ctx, id)
}

func (s *RemnaService) GetUserByEmail(ctx context.Context, email string) (*RemnaUserResponse, error) {
	return s.client.GetUserByEmail(ctx, email)
}

func (s *RemnaService) GetUserByTelegramID(ctx context.Context, tgID string) (*RemnaUserResponse, error) {
	return s.client.GetUserByTelegramID(ctx, tgID)
}

func (s *RemnaService) CreateNewUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error) {
	return s.client.CreateNewUser(ctx, user)
}

func (s *RemnaService) UpdateUser(ctx context.Context, user *RemnaUserRequest) (*RemnaUserResponse, error) {
	return s.client.UpdateUser(ctx, user)
}
