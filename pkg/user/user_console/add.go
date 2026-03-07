package user_console

import (
	"encoding/json"
	"fmt"

	"github.com/evgeniums/evgo/pkg/console_tool"
	"github.com/evgeniums/evgo/pkg/user"
)

const AddCmd string = "add"
const AddDescription string = "Add new"

type LoginData struct {
	Login string `long:"login" description:"Login" required:"true"`
}

type PhoneData struct {
	Phone string `json:"phone" long:"phone" description:"Phone number for SMS confirmations" required:"true" validate:"omitempty,phone" vmessage:"Invalid phone format"`
}

type EmailData struct {
	Email string `json:"email" long:"email" description:"Email address" required:"true" validate:"omitempty,email" vmessage:"Invalid email format"`
}

type AddData struct {
	LoginData
}

type WithPhoneData struct {
	LoginData
	PhoneData
}

type WithEmailData struct {
	LoginData
	EmailData
}

//----------------------------------------

func Add[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &AddHandler[T]{}
	a.Init(AddCmd, AddDescription)
	return a
}

type AddHandler[T user.User] struct {
	HandlerBase[T]
	AddData
}

func (a *AddHandler[T]) Data() interface{} {
	return &a.AddData
}

func (a *AddHandler[T]) Execute(args []string) error {

	ctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close()

	password := console_tool.ReadPassword()
	user, err := ctrl.Add(ctx, a.Login, password)
	if err != nil {
		return err
	}
	return dumpUser(user)
}

//----------------------------------------

func AddNoPassword[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &AddNoPasswordHandler[T]{}
	a.Init(AddCmd, AddDescription)
	return a
}

type AddNoPasswordHandler[T user.User] struct {
	HandlerBase[T]
	AddData
}

func (a *AddNoPasswordHandler[T]) Data() interface{} {
	return &a.AddData
}

func (a *AddNoPasswordHandler[T]) Execute(args []string) error {

	ctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close()

	user, err := ctrl.Add(ctx, a.Login, "00000000")
	if err != nil {
		return err
	}
	return dumpUser(user)
}

//----------------------------------------

func AddWithPhone[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &AddWithPhoneHandler[T]{}
	a.Init(AddCmd, AddDescription)
	return a
}

type AddWithPhoneHandler[T user.User] struct {
	HandlerBase[T]
	WithPhoneData
}

func (a *AddWithPhoneHandler[T]) Data() interface{} {
	return &a.WithPhoneData
}

func (a *AddWithPhoneHandler[T]) Execute(args []string) error {

	ctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close()

	password := console_tool.ReadPassword()
	user, err := ctrl.Add(ctx, a.Login, password, user.Phone[T](a.Phone))
	if err != nil {
		return err
	}
	return dumpUser(user)
}

//----------------------------------------

func AddWithEmail[T user.User]() console_tool.Handler[*UserCommands[T]] {
	a := &AddWithEmailHandler[T]{}
	a.Init(AddCmd, AddDescription)
	return a
}

type AddWithEmailHandler[T user.User] struct {
	HandlerBase[T]
	WithEmailData
}

func (a *AddWithEmailHandler[T]) Data() interface{} {
	return &a.WithEmailData
}

func (a *AddWithEmailHandler[T]) Execute(args []string) error {

	ctx, ctrl, err := a.Context(a.Data(), a.Login)
	if err != nil {
		return err
	}
	defer ctx.Close()

	password := console_tool.ReadPassword()
	user, err := ctrl.Add(ctx, a.Login, password, user.Email[T](a.Email))
	if err != nil {
		return err
	}
	return dumpUser(user)
}

func dumpUser[T user.User](user T) error {
	b, err := json.MarshalIndent(user, "", "   ")
	if err != nil {
		return fmt.Errorf("failed to serialize result: %s", err)
	}
	fmt.Printf("Created object:\n\n%s\n\n", string(b))
	return nil
}
