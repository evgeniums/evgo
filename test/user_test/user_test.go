package user_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_default"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

type User = user_default.User

func dbModels() []interface{} {
	return append([]interface{}{}, &User{}, &user.OpLogUser{})
}

func initTest(t *testing.T, config ...string) (app_context.Context, *user_default.Users, op_context.Context) {
	app := test_utils.InitAppContext(t, testDir, dbModels(), utils.OptionalArg("user_test.json", config...))

	users := user_default.NewUsers()
	users.Init(app.Validator())

	ctx := test_utils.SimpleOpContext(app, t.Name())

	return app, users, ctx
}

func TestInitUsers(t *testing.T) {
	app, _, _ := initTest(t)
	app.Close()
}

func TestUserOperations(t *testing.T) {
	app, users, ctx := initTest(t)
	onExit := func() {
		ctx.Close()
		app.Close()
	}
	defer onExit()

	login1 := "user1"
	password1 := "password1"
	phone1 := "1122334455"
	email1 := "user1@example.com"
	user1, err := users.Add(ctx, login1, password1, user.Phone(phone1, &User{}), user.Email(email1, &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)
	assert.Equal(t, login1, user1.Login())
	assert.Equal(t, phone1, user1.Phone())
	assert.Equal(t, email1, user1.Email())

	login2 := "user2"
	password2 := "password2"
	phone2 := "8822334477"
	email2 := "user2@example.com"
	user2, err := users.Add(ctx, login2, password2, user.Phone(phone2, &User{}), user.Email(email2, &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user2)
	assert.Equal(t, login2, user2.Login())
	assert.Equal(t, phone2, user2.Phone())
	assert.Equal(t, email2, user2.Email())

	userDb1_1, err := users.FindByLogin(ctx, login1)
	require.NoErrorf(t, err, "failed to find user")
	require.NotNil(t, userDb1_1)
	assert.True(t, user1.GetCreatedAt().Equal(userDb1_1.GetCreatedAt()))
	assert.True(t, user1.GetUpdatedAt().Equal(userDb1_1.GetUpdatedAt()))
	userDb1_1.SetCreatedAt(user1.GetCreatedAt())
	userDb1_1.SetUpDatedAt(user1.GetUpdatedAt())
	assert.Equal(t, user1, userDb1_1)

	userNotInDb, err := users.FindByLogin(ctx, "unknown-login")
	require.Error(t, err)
	require.Nil(t, userNotInDb)

	newPhone := "999000111"
	require.NoError(t, users.SetPhone(ctx, login1, newPhone, true))
	userDb1_2, err := users.FindByLogin(ctx, login1)
	require.NoErrorf(t, err, "failed to find user")
	require.NotNil(t, userDb1_2)
	assert.Equal(t, newPhone, userDb1_2.Phone())
	assert.Equal(t, email1, userDb1_2.Email())

	newEmail := "user1_1@example.com"
	require.NoError(t, users.SetEmail(ctx, login1, newEmail, true))
	userDb1_3, err := users.FindByLogin(ctx, login1)
	require.NoErrorf(t, err, "failed to find user")
	require.NotNil(t, userDb1_3)
	assert.Equal(t, newPhone, userDb1_3.Phone())
	assert.Equal(t, newEmail, userDb1_3.Email())

	require.NoError(t, users.SetBlocked(ctx, login1, true, true))
	userDb1_4, err := users.FindByLogin(ctx, login1)
	require.NoErrorf(t, err, "failed to find user")
	require.NotNil(t, userDb1_4)
	assert.True(t, userDb1_4.IsBlocked())

	newPassword := "bla-bla-new"
	require.NoError(t, users.SetPassword(ctx, login1, newPassword, true))
	userDb1_5, err := users.FindByLogin(ctx, login1)
	require.NoErrorf(t, err, "failed to find user")
	require.NotNil(t, userDb1_5)
}
