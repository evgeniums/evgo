package user

import (
	"errors"
	"fmt"

	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const ErrorCodeDuplicateLogin string = "duplicate_login"
const ErrorCodeDuplicateEmail string = "duplicate_email"
const ErrorCodeDuplicatePhone string = "duplicate_phone"

var ErrorDescriptions = map[string]string{
	ErrorCodeDuplicateLogin: "Login already occupied",
	ErrorCodeDuplicateEmail: "Email address is already in use",
	ErrorCodeDuplicatePhone: "Phone number is already in use",
}

var ErrorHttpCodes = map[string]int{}

type MainFieldSetters interface {
	SetPassword(ctx op_context.Context, id string, password string, idIsLogin ...bool) error
	SetPhone(ctx op_context.Context, id string, phone string, idIsLogin ...bool) error
	SetEmail(ctx op_context.Context, id string, email string, idIsLogin ...bool) error
	SetBlocked(ctx op_context.Context, id string, blocked bool, idIsLogin ...bool) error
}

type UserFinder[U User] interface {
	Find(ctx op_context.Context, id string) (U, error)
	FindByLogin(ctx op_context.Context, login string) (U, error)
}

type UserController[UserType User] interface {
	MainFieldSetters
	UserFinder[UserType]

	Add(ctx op_context.Context, login string, password string, extraFieldsSetters ...SetUserFields[UserType]) (UserType, error)
	FindUsers(ctx op_context.Context, filter *db.Filter) ([]UserType, int64, error)

	SetUserBuilder(builder func() UserType)
	MakeUser() UserType

	SetOplogBuilder(builder func() OpLogUserI)
	OpLog(ctx op_context.Context, op string, userId string, login string)
}

type Users[UserType User] interface {
	auth_session.AuthUserManager
	UserController[UserType]
}

type UserControllerBase[UserType User] struct {
	userBuilder    func() UserType
	oplogBuilder   func() OpLogUserI
	crudController crud.CRUD
	userValidators auth_session.UserValidators
}

func LocalUserController[UserType User]() *UserControllerBase[UserType] {
	return &UserControllerBase[UserType]{crudController: &crud.DbCRUD{}, oplogBuilder: func() OpLogUserI { return &OpLogUser{} }}
}

func (u *UserControllerBase[UserType]) SetUserBuilder(userBuilder func() UserType) {
	u.userBuilder = userBuilder
}

func (u *UserControllerBase[UserType]) SetOplogBuilder(builder func() OpLogUserI) {
	u.oplogBuilder = builder
}

func (u *UserControllerBase[UserType]) SetUserValidators(validators auth_session.UserValidators) {
	u.userValidators = validators
}

func (u *UserControllerBase[UserType]) CRUD() crud.CRUD {
	return u.crudController
}

func (u *UserControllerBase[UserType]) MakeUser() UserType {
	return u.userBuilder()
}

func (u *UserControllerBase[UserType]) Add(ctx op_context.Context, login string, password string, extraFieldsSetters ...SetUserFields[UserType]) (UserType, error) {

	// setup
	ctx.SetLoggerField("login", login)
	c := ctx.TraceInMethod("Users.Add")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// validate
	if u.userValidators != nil {
		err = u.userValidators.ValidateLogin(login)
		if err != nil {
			c.SetMessage("failed to validate login")
			return *new(UserType), err
		}
		err = u.userValidators.ValidatePassword(password)
		if err != nil {
			c.SetMessage("failed to validate password")
			return *new(UserType), err
		}
	}

	// check if user with such login already exists
	filter := db.NewFilter()
	filter.AddField("login", login)
	filter.Limit = 1
	users, _, err := u.FindUsers(ctx, filter)
	if err != nil {
		c.SetMessage("failed to check login duplicates")
		return *new(UserType), err
	}
	if len(users) > 0 {
		err = errors.New("duplicate login")
		ctx.SetGenericErrorCode(ErrorCodeDuplicateLogin)
		return *new(UserType), err
	}

	// create user
	user := u.MakeUser()
	user.InitObject()
	user.SetLogin(login)
	user.SetPassword(password)
	for _, setter := range extraFieldsSetters {
		checkDuplicateFields, err1 := setter(ctx, user)
		err = err1
		if err != nil {
			c.SetMessage("failed to set extra fields")
			return *new(UserType), err
		}
		for _, checkDup := range checkDuplicateFields {
			filter := db.NewFilter()
			filter.AddField(checkDup.Name, checkDup.Value)
			filter.Limit = 1
			users, _, err1 := u.FindUsers(ctx, filter)
			err = err1
			if err != nil {
				c.SetLoggerField("unique_field", checkDup.Name)
				c.SetMessage("failed to check unique field")
				return *new(UserType), err
			}
			if len(users) > 0 {
				c.SetLoggerField("unique_field", checkDup.Name)
				c.SetLoggerField("unique_value", checkDup.Value)
				err = fmt.Errorf("duplicate %s", checkDup.Name)
				ctx.SetGenericErrorCode(checkDup.ErrorCode)
				return *new(UserType), err
			}
		}
	}

	// create in manager
	err = u.crudController.Create(ctx, user)
	if err != nil {
		c.SetMessage("failed to create user")
		return *new(UserType), err
	}
	u.OpLog(ctx, "add", user.GetID(), login)

	// done
	return user, nil
}

func (u *UserControllerBase[UserType]) Find(ctx op_context.Context, id string) (UserType, error) {

	// setup
	c := ctx.TraceInMethod("Users.Find", logger.Fields{"id": id})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user := u.MakeUser()
	found, err := u.crudController.Read(ctx, db.Fields{"id": id}, user)
	if err != nil {
		c.SetMessage("failed to find user in database")
		return *new(UserType), err
	}
	if !found {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
		err = errors.New("user with such ID does not exist")
		return *new(UserType), err
	}

	// done
	return user, nil
}

func (u *UserControllerBase[UserType]) FindByLogin(ctx op_context.Context, login string) (UserType, error) {

	// setup
	c := ctx.TraceInMethod("Users.FindByLogin", logger.Fields{"login": login})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user := u.MakeUser()
	found, err := FindByLogin(u.crudController, ctx, login, user)
	if err != nil {
		c.SetMessage("failed to find user in database")
		return *new(UserType), err
	}
	if !found {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
		err = errors.New("user with such login does not exist")
		return *new(UserType), err
	}

	// done
	return user, nil
}

func FindUser[UserType User](u UserFinder[UserType], ctx op_context.Context, id string, idIsLogin ...bool) (user UserType, err error) {

	useLogin := utils.OptionalArg(false, idIsLogin...)

	if useLogin {
		ctx.SetLoggerField("login", id)
		user, err = u.FindByLogin(ctx, id)
	} else {
		ctx.SetLoggerField("id", id)
		user, err = u.Find(ctx, id)
	}

	return
}

func (u *UserControllerBase[UserType]) SetPassword(ctx op_context.Context, id string, password string, idIsLogin ...bool) error {

	// setup
	c := ctx.TraceInMethod("Users.SetPassword")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// validate
	if u.userValidators != nil {
		err = u.userValidators.ValidatePassword(password)
		if err != nil {
			c.SetMessage("failed to validate password")
			return err
		}
	}

	// find user
	user, err := FindUser[UserType](u, ctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set password
	user.SetPassword(password)
	err = u.crudController.Update(ctx, user, db.Fields{"password_hash": user.PasswordHash(), "password_salt": user.PasswordSalt()})
	if err != nil {
		return err
	}

	// done
	u.OpLog(ctx, "set_password", user.GetID(), user.Login())
	return nil
}

func (u *UserControllerBase[UserType]) SetPhone(ctx op_context.Context, id string, phone string, idIsLogin ...bool) error {

	// setup
	ctx.SetLoggerField("phone", phone)
	c := ctx.TraceInMethod("Users.SetPhone")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := FindUser[UserType](u, ctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set phone
	err = u.crudController.Update(ctx, user, db.Fields{"phone": phone})
	if err != nil {
		return err
	}

	// done
	u.OpLog(ctx, "set_phone", user.GetID(), user.Login())
	return nil
}

func (u *UserControllerBase[UserType]) SetEmail(ctx op_context.Context, id string, email string, idIsLogin ...bool) error {

	// setup
	ctx.SetLoggerField("email", email)
	c := ctx.TraceInMethod("Users.SetEmail")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := FindUser[UserType](u, ctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set email
	err = u.crudController.Update(ctx, user, db.Fields{"email": email})
	if err != nil {
		return err
	}

	// done
	u.OpLog(ctx, "set_email", user.GetID(), user.Login())
	return nil
}

func (u *UserControllerBase[UserType]) SetBlocked(ctx op_context.Context, id string, blocked bool, idIsLogin ...bool) error {

	// setup
	ctx.SetLoggerField("blocked", blocked)
	c := ctx.TraceInMethod("Users.SetBlocked")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find user
	user, err := FindUser[UserType](u, ctx, id, idIsLogin...)
	if err != nil {
		return err
	}

	// set blocked
	err = u.crudController.Update(ctx, user, db.Fields{"blocked": blocked})
	if err != nil {
		return err
	}

	// done
	u.OpLog(ctx, "set_blocked", user.GetID(), user.Login())
	return nil
}

func (u *UserControllerBase[UserType]) FindUsers(ctx op_context.Context, filter *db.Filter) ([]UserType, int64, error) {
	var users []UserType
	count, err := u.crudController.List(ctx, filter, &users)
	return users, count, err
}

func (u *UserControllerBase[UserType]) OpLog(ctx op_context.Context, op string, userId string, login string) {
	oplog := u.oplogBuilder()
	oplog.SetOperation(op)
	oplog.SetLogin(login)
	oplog.SetUserId(userId)
	ctx.Oplog(oplog)
}

type UsersValidator struct {
	Validator            validator.Validator
	LoginValidationRules string
}

func (u *UsersValidator) ValidateLogin(login string) error {
	err := u.Validator.ValidateValue(login, u.LoginValidationRules)
	if err != nil {
		return &validator.ValidationError{Message: "Invalid format of login", Field: "login"}
	}
	return nil
}

func (u *UsersValidator) ValidatePassword(password string) error {
	if len(password) < 8 {
		return &validator.ValidationError{Message: "Password must be at least 8 characters", Field: "password"}
	}
	return nil
}

func (u *UsersValidator) Init(vld validator.Validator, loginValidationRules ...string) {
	u.Validator = vld
	u.LoginValidationRules = utils.OptionalArg("required,alphanum_|email,lowercase", loginValidationRules...)
}

type UsersBase[UserType User] struct {
	UsersValidator
	UserController[UserType]
	auth_session.AuthUserFinder
}

func (u *UsersBase[UserType]) Construct(userController UserController[UserType], authUserFinder ...auth_session.AuthUserFinder) {
	u.UserController = userController
	if len(authUserFinder) != 0 {
		u.AuthUserFinder = authUserFinder[0]
	} else {
		u.AuthUserFinder = NewAuthUserFinder(func() User { return u.UserController.MakeUser() })
	}
}

func (m *UsersBase[UserType]) AuthUserManager() auth_session.AuthUserManager {
	return m
}

func (u *UsersBase[UserType]) SetAuthUserFinder(authUserFinder auth_session.AuthUserFinder) {
	u.AuthUserFinder = authUserFinder
}
