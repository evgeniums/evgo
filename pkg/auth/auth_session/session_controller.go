package auth_session

import (
	"time"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type SessionController interface {
	CreateSession(ctx auth.ContextWithAuthUser, expiration time.Time) (Session, error)
	FindSession(ctx op_context.Context, sessionId string) (Session, error)
	UpdateSessionClient(ctx auth.AuthContext) error
	UpdateSessionExpiration(ctx auth.AuthContext, session Session) error
	InvalidateSession(ctx op_context.Context, userId string, sessionId string) error
	InvalidateUserSessions(ctx op_context.Context, userId string) error
	InvalidateAllSessions(ctx op_context.Context) error

	GetSessions(ctx op_context.Context, filter *db.Filter, sessions interface{}) (int64, error)
	GetSessionClients(ctx op_context.Context, filter *db.Filter, sessionClients interface{}) (int64, error)

	SetSessionBuilder(func() Session)
	MakeSession() Session
	SetSessionClientBuilder(func() SessionClient)
	MakeSessionClient() SessionClient
}

type SessionControllerBase struct {
	sessionBuilder       func() Session
	sessionClientBuilder func() SessionClient
	crud                 crud.CRUD
}

func LocalSessionController(cr ...crud.CRUD) *SessionControllerBase {
	s := &SessionControllerBase{}
	if len(cr) == 0 {
		s.crud = &crud.DbCRUD{}
	}
	return s
}

func (s *SessionControllerBase) SetSessionBuilder(sessionBuilder func() Session) {
	s.sessionBuilder = sessionBuilder
}

func (s *SessionControllerBase) SetSessionClientBuilder(sessionClientBuilder func() SessionClient) {
	s.sessionClientBuilder = sessionClientBuilder
}

func (s *SessionControllerBase) MakeSession() Session {
	return s.sessionBuilder()
}

func (s *SessionControllerBase) MakeSessionClient() SessionClient {
	return s.sessionClientBuilder()
}

func (s *SessionControllerBase) CreateSession(ctx auth.ContextWithAuthUser, expiration time.Time) (Session, error) {

	c := ctx.TraceInMethod("auth_session.CreateSession")
	defer ctx.TraceOutMethod()

	session := s.MakeSession()
	session.InitObject()
	session.SetUser(ctx.AuthUser())
	session.SetValid(true)
	session.SetExpiration(expiration)

	err := s.crud.Create(ctx, session)
	if err != nil {
		return nil, c.SetError(err)
	}

	return session, nil
}

func (s *SessionControllerBase) FindSession(ctx op_context.Context, sessionId string) (Session, error) {

	c := ctx.TraceInMethod("auth_session.FindSession")
	defer ctx.TraceOutMethod()

	session := s.MakeSession()
	_, err := s.crud.ReadByField(ctx, "id", sessionId, session)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return nil, c.SetError(err)
	}

	return session, nil
}

func (s *SessionControllerBase) UpdateSessionClient(ctx auth.AuthContext) error {

	// setup
	c := ctx.TraceInMethod("auth_session.UpdateSessionClient")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// extract client parameters
	clientIp := ctx.GetRequestClientIp()
	userAgent := ctx.GetRequestUserAgent()
	h := crypt_utils.NewHash()
	clientHash := h.CalcStrStr(clientIp, userAgent)

	// find client in database
	tryUpdate := true
	client := s.MakeSessionClient()
	fields := db.Fields{"session_id": ctx.GetSessionId(), "client_hash": clientHash}
	found, err := s.crud.Read(ctx, fields, client)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		c.SetMessage("failed to find client in database")
		return err
	}
	if !found {
		// create new client
		tryUpdate = false
		client.InitObject()
		client.SetClientIp(clientIp)
		client.SetClientHash(clientHash)
		client.SetSessionId(ctx.GetSessionId())
		client.SetUser(ctx.AuthUser())
		client.SetUserAgent(userAgent)
		err1 := s.crud.Create(ctx, client)
		if err1 != nil {
			c.Logger().Error("failed to create session client in database", err1)
			tryUpdate = true
		}
	}

	// update client
	if tryUpdate {
		err = s.crud.Update(ctx, client, db.Fields{"updated_at": time.Now()})
		if err != nil {
			ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
			c.SetMessage("failed to update client in database")
			return err
		}
	}

	ctx.SetClientId(client.GetID())
	ctx.SetLoggerField("client", client.GetID())
	return nil
}

func (s *SessionControllerBase) UpdateSessionExpiration(ctx auth.AuthContext, session Session) error {

	c := ctx.TraceInMethod("auth_session.UpdateSessionExpiration")
	defer ctx.TraceOutMethod()

	err := s.crud.Update(ctx, session, db.Fields{"expiration": session.GetExpiration()})
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return c.SetError(err)
	}
	return nil
}

func (s *SessionControllerBase) InvalidateSession(ctx op_context.Context, userId string, sessionId string) error {

	c := ctx.TraceInMethod("auth_session.InvalidateSession")
	defer ctx.TraceOutMethod()

	err := s.crud.UpdateMulti(ctx, s.MakeSession(), db.Fields{"id": sessionId, "user_id": userId}, db.Fields{"valid": false, "updated_at": time.Now()})
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return c.SetError(err)
	}
	return nil

}

func (s *SessionControllerBase) InvalidateUserSessions(ctx op_context.Context, userId string) error {
	c := ctx.TraceInMethod("auth_session.InvalidateUserSessions")
	defer ctx.TraceOutMethod()

	err := s.crud.UpdateMulti(ctx, s.MakeSession(), db.Fields{"user_id": userId}, db.Fields{"valid": false, "updated_at": time.Now()})
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return c.SetError(err)
	}
	return nil
}

func (s *SessionControllerBase) InvalidateAllSessions(ctx op_context.Context) error {
	c := ctx.TraceInMethod("auth_session.InvalidateAllSessions")
	defer ctx.TraceOutMethod()

	err := s.crud.UpdateMulti(ctx, s.MakeSession(), nil, db.Fields{"valid": false, "updated_at": time.Now()})
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return c.SetError(err)
	}
	return nil
}

// Get sessions using filter. Note that sessions argument must be of *[]Session type.
func (s *SessionControllerBase) GetSessions(ctx op_context.Context, filter *db.Filter, sessions interface{}) (int64, error) {

	c := ctx.TraceInMethod("auth_session.GetSessions")
	defer ctx.TraceOutMethod()
	count, err := s.crud.List(ctx, filter, sessions)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return 0, c.SetError(err)
	}
	return count, nil
}

// Get sessions using filter. Note that sessions argument must be of *[]SessionClient type.
func (s *SessionControllerBase) GetSessionClients(ctx op_context.Context, filter *db.Filter, sessions interface{}) (int64, error) {

	c := ctx.TraceInMethod("auth_session.GetSessionClients")
	defer ctx.TraceOutMethod()
	count, err := s.crud.List(ctx, filter, sessions)
	if err != nil {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return 0, c.SetError(err)
	}
	return count, nil
}
