package gomail_sender

import (
	"context"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/email_sender"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	gomail "gopkg.in/mail.v2"
)

type GomailSenderConfig struct {
	HOST         string `validate:"required" vmessage:"Gomail client host must be specified"`
	PORT         uint16 `validate:"required" vmessage:"Gomail client port must be specified"`
	USER         string `validate:"required"  vmessage:"Gomail client user must be specified"`
	PASSWORD     string `validate:"required"  vmessage:"Gomail client password must be specified"`
	FROM_ADDRESS string
	FROM_NAME    string
}

type GomailSender struct {
	GomailSenderConfig

	dialer *gomail.Dialer
}

func (g *GomailSender) Config() interface{} {
	return &g.GomailSenderConfig
}

func (g *GomailSender) Init(app app_context.Context, configPath ...string) error {

	path := utils.OptionalString("mailer", configPath...)
	err := object_config.LoadLogValidateApp(app, g, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of mailer", err)
	}
	if g.FROM_ADDRESS == "" {
		g.FROM_ADDRESS = g.USER
	}

	g.dialer = gomail.NewDialer(g.HOST, int(g.PORT), g.USER, g.PASSWORD)

	return nil
}

func (g *GomailSender) prepareMessage(to string, subject string) *gomail.Message {

	m := gomail.NewMessage()

	m.SetAddressHeader("From", g.FROM_ADDRESS, g.FROM_NAME)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	return m
}

func (g *GomailSender) Send(sctx context.Context, to string, subject string, contents ...email_sender.EmailContent) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("GomailSender.Send", logger.Fields{"to": to})
	defer ctx.TraceOutMethod()

	// prepare message
	m := g.prepareMessage(to, subject)

	// load content
	i := 0
	for _, content := range contents {
		if i == 0 {
			m.SetBody(content.ContentType, content.Content)
		} else {
			m.AddAlternative(content.ContentType, content.Content)
		}
		i++
	}

	// dial and send
	err := g.dialer.DialAndSend(m)
	if err != nil {
		c.SetMessage("failed to send email")
		return c.SetError(err)
	}

	// done
	return nil
}

func New() *GomailSender {
	return &GomailSender{}
}
