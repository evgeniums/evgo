package auth_session

import (
	"time"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/common"
)

type Session interface {
	common.Object
	auth.WithUser

	SetValid(valid bool)
	SetExpiration(exp time.Time)

	IsValid() bool
	GetExpiration() time.Time
}

type SessionBase struct {
	common.ObjectBase
	auth.WithUserBase
	Valid      bool      `gorm:"index"`
	Expiration time.Time `gorm:"index"`
}

func NewSession() Session {
	return &SessionBase{}
}

func (s *SessionBase) SetValid(valid bool) {
	s.Valid = valid
}

func (s *SessionBase) IsValid() bool {
	return s.Valid
}

func (s *SessionBase) SetExpiration(exp time.Time) {
	s.Expiration = exp
}

func (s *SessionBase) GetExpiration() time.Time {
	return s.Expiration
}

type SessionClient interface {
	common.Object
	auth.WithUser
	SetSessionId(userId string)
	SetClientIp(clientIp string)
	SetUserAgent(userAgent string)
	SetClientHash(clientHash string)

	GetSessionId() string
	GetClientIp() string
	GetUserAgent() string
	GetClientHash() string
}

type SessionClientBase struct {
	common.ObjectBase
	auth.WithUserBase
	SessionId  string `gorm:"index;index:,unique,composite:client_session"`
	ClientIp   string `gorm:"index"`
	UserAgent  string
	ClientHash string `gorm:"index;index:,unique,composite:client_session"`
}

func NewSessionClient() SessionClient {
	return &SessionClientBase{}
}

func (s *SessionClientBase) SetSessionId(sessionId string) {
	s.SessionId = sessionId
}

func (s *SessionClientBase) GetSessionId() string {
	return s.SessionId
}

func (s *SessionClientBase) SetClientIp(val string) {
	s.ClientIp = val
}

func (s *SessionClientBase) GetClientIp() string {
	return s.ClientIp
}

func (s *SessionClientBase) SetUserAgent(val string) {
	s.UserAgent = val
}

func (s *SessionClientBase) GetUserAgent() string {
	return s.UserAgent
}

func (s *SessionClientBase) SetClientHash(val string) {
	s.ClientHash = val
}

func (s *SessionClientBase) GetClientHash() string {
	return s.ClientHash
}
