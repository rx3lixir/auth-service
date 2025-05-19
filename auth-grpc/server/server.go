package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authPb "github.com/rx3lixir/auth-service/auth-grpc/gen/go"
	"github.com/rx3lixir/auth-service/internal/db"
)

type Server struct {
	storer *db.RedisStore
	authPb.UnsafeAuthServiceServer
}

func NewServer(storer *db.RedisStore) *Server {
	return &Server{
		storer: storer,
	}
}

// CreateSession создает новую сессию
func (s *Server) CreateSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	session := ConvertProtoToSession(req)

	// Проверка обязательных полей
	if session.UserEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "user_email is required")
	}

	if session.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	session.ExpiresAt = time.Now().Add(time.Minute * 15)

	createdSession, err := s.storer.CreateSession(ctx, session)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}
	return ConvertSessionToProto(createdSession), nil
}

// GetSession получает сессию по ID
func (s *Server) GetSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	return ConvertSessionToProto(session), nil
}

// Revoke session отзывает сессию
func (s *Server) RevokeSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	// Перед отзывом получаем текущую сессию
	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	if err := s.storer.RevokeSession(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke session: %v", err)
	}

	// Обновляем статус сессии в ответе
	session.IsRevoked = true

	return ConvertSessionToProto(session), nil
}

// DeleteSession удаляет сессию
func (s *Server) DeleteSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	if err := s.storer.DeleteSession(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", err)
	}

	return ConvertSessionToProto(session), nil
}
