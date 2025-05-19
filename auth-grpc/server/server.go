package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authPb "github.com/rx3lixir/auth-service/auth-grpc/gen/go"
	"github.com/rx3lixir/auth-service/internal/config"
	"github.com/rx3lixir/auth-service/internal/db"
	"github.com/rx3lixir/auth-service/internal/logger"
)

type Server struct {
	storer *db.RedisStore
	authPb.UnsafeAuthServiceServer
	log  logger.Logger
	conf *config.AppConfig
}

func NewServer(storer *db.RedisStore, log logger.Logger, config *config.AppConfig) *Server {
	return &Server{
		storer: storer,
		log:    log,
		conf:   config,
	}
}

// CreateSession создает новую сессию
func (s *Server) CreateSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	s.log.Info("starting create session",
		"method", "CreateSession",
		"user_email", req.UserEmail,
		"session_id", req.Id,
	)

	session := ConvertProtoToSession(req)

	if session.UserEmail == "" {
		s.log.Error("missing required field",
			"method", "CreateSession",
			"missing_field", "user_email",
		)
		return nil, status.Error(codes.InvalidArgument, "user_email is required")
	}

	if session.Id == "" {
		s.log.Error("missing required field",
			"method", "CreateSession",
			"missing_field", "session_id",
		)
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Устанавливаем время истечения из конфигурации
	session.ExpiresAt = time.Now().Add(s.conf.Service.GetSessionTTL())

	createdSession, err := s.storer.CreateSession(ctx, session)
	if err != nil {
		s.log.Error("failed to create session",
			"method", "CreateSession",
			"error", err,
			"user_email", req.UserEmail,
			"session_id", req.Id,
		)
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	s.log.Info("session created successfully",
		"method", "CreateSession",
		"session_id", createdSession.Id,
		"user_email", createdSession.UserEmail,
		"expires_at", createdSession.ExpiresAt,
	)
	return ConvertSessionToProto(createdSession), nil
}

// GetSession получает сессию по ID
func (s *Server) GetSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	s.log.Info("starting get session",
		"method", "GetSession",
		"session_id", req.Id,
	)

	if req.Id == "" {
		s.log.Error("missing required field",
			"method", "GetSession",
			"missing_field", "session_id",
		)
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		s.log.Error("session not found",
			"method", "GetSession",
			"session_id", req.Id,
			"error", err,
		)
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	s.log.Info("session retrieved successfully",
		"method", "GetSession",
		"session_id", session.Id,
		"is_revoked", session.IsRevoked,
		"expires_in", time.Until(session.ExpiresAt),
	)
	return ConvertSessionToProto(session), nil
}

// RevokeSession отзывает сессию
func (s *Server) RevokeSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	s.log.Info("starting revoke session",
		"method", "RevokeSession",
		"session_id", req.Id,
	)

	if req.Id == "" {
		s.log.Error("missing required field",
			"method", "RevokeSession",
			"missing_field", "session_id",
		)
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		s.log.Error("session not found for revoke",
			"method", "RevokeSession",
			"session_id", req.Id,
			"error", err,
		)
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	if err := s.storer.RevokeSession(ctx, req.Id); err != nil {
		s.log.Error("failed to revoke session",
			"method", "RevokeSession",
			"session_id", req.Id,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to revoke session: %v", err)
	}

	s.log.Info("session revoked successfully",
		"method", "RevokeSession",
		"session_id", req.Id,
	)
	session.IsRevoked = true

	return ConvertSessionToProto(session), nil
}

// DeleteSession удаляет сессию
func (s *Server) DeleteSession(ctx context.Context, req *authPb.SessionReq) (*authPb.SessionRes, error) {
	s.log.Info("starting delete session",
		"method", "DeleteSession",
		"session_id", req.Id,
	)

	if req.Id == "" {
		s.log.Error("missing required field",
			"method", "DeleteSession",
			"missing_field", "session_id",
		)
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	session, err := s.storer.GetSession(ctx, req.Id)
	if err != nil {
		s.log.Error("session not found for delete",
			"method", "DeleteSession",
			"session_id", req.Id,
			"error", err,
		)
		return nil, status.Errorf(codes.NotFound, "session not found: %v", err)
	}

	if err := s.storer.DeleteSession(ctx, req.Id); err != nil {
		s.log.Error("failed to delete session",
			"method", "DeleteSession",
			"session_id", req.Id,
			"error", err,
		)
		return nil, status.Errorf(codes.Internal, "failed to delete session: %v", err)
	}

	s.log.Info("session deleted successfully",
		"method", "DeleteSession",
		"session_id", req.Id,
		"user_email", session.UserEmail,
	)
	return ConvertSessionToProto(session), nil
}
