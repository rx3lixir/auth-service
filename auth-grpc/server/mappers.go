package server

import (
	"time"

	authPb "github.com/rx3lixir/auth-service/auth-grpc/gen/go"
	"github.com/rx3lixir/auth-service/internal/db"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConvertSessionToProto преобразует внутреннюю модель Session в protobuf SessionRes
func ConvertSessionToProto(session *db.Session) *authPb.SessionRes {
	if session == nil {
		return nil
	}

	return &authPb.SessionRes{
		Id:           session.Id,
		UserEmail:    session.UserEmail,
		RefreshToken: session.RefreshToken,
		IsRevoked:    session.IsRevoked,
		ExpiresAt:    timestamppb.New(session.ExpiresAt),
	}
}

// ConvertProtoToSession преобразует protobuf SessionReq во внутреннюю модель Session
func ConvertProtoToSession(sessionReq *authPb.SessionReq) *db.Session {
	if sessionReq == nil {
		return nil
	}

	var expiresAt time.Time
	if sessionReq.ExpiresAt != nil {
		expiresAt = sessionReq.ExpiresAt.AsTime()
	}

	return &db.Session{
		Id:           sessionReq.Id,
		UserEmail:    sessionReq.UserEmail,
		RefreshToken: sessionReq.RefreshToken,
		IsRevoked:    sessionReq.IsRevoked,
		ExpiresAt:    expiresAt,
	}
}
