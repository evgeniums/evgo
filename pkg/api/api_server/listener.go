package api_server

import (
	"fmt"
	"net"
	"strings"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
)

type ListenerConfig struct {
	common.WithNameBaseConfig
	HOST string `validate:"ip" default:"127.0.0.1"`
	PORT uint16 `validate:"required"`

	TLS_CERTIFICATE_FILE string
	TLS_PRIVATE_KEY_FILE string
	DISABLE_TLS          bool
}

type Listener struct {
	app_context.WithAppBase
	ListenerConfig

	listener net.Listener
}

func (l *Listener) Config() any {
	return &l.ListenerConfig
}

func (l *Listener) Init(ctx app_context.Context, configPath ...string) error {

	var err error

	l.WithAppBase.Init(ctx)

	err = object_config.LoadLogValidate(ctx.Cfg(), ctx.Logger(), ctx.Validator(), l, "listener", configPath...)
	if err != nil {
		return ctx.Logger().PushFatalStack("failed to load listener configuration", err)
	}

	return nil
}

func (l *Listener) Run(protocol ...string) {
	var err error
	l.listener, err = net.Listen(utils.OptionalString("tcp", protocol...), l.Address())
	if err != nil {
		msg := "TCP listening failed"
		l.App().Logger().Fatal(msg, err, logger.Fields{"name": l.Name()})
		app_context.AbortFatal(l.App(), msg)
	}

	l.App().Logger().Info("Listening for incoming connections", logger.Fields{"address": l.listener.Addr().String()})
}

func (l *Listener) Address() string {
	if strings.Contains(l.HOST, "::") {
		return fmt.Sprintf("[%s]:%d", l.HOST, l.PORT)
	}
	return fmt.Sprintf("%s:%d", l.HOST, l.PORT)
}

func (l *Listener) Listener() net.Listener {
	return l.listener
}
