package email_sender

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type EmailContent struct {
	ContentType string
	Content     string
}

type TemplateVals = map[string]string

type EmailSender interface {
	Send(ctx op_context.Context, to string, subject string, content ...EmailContent) error
}

func SendText(ctx op_context.Context, client EmailSender, to string, subject string, content string) error {
	return client.Send(ctx, to, subject, EmailContent{ContentType: "text/plain", Content: content})
}

func SendHtml(ctx op_context.Context, client EmailSender, to string, subject string, content string) error {
	return client.Send(ctx, to, subject, EmailContent{ContentType: "text/html", Content: content})
}

func SendHtmlAndText(ctx op_context.Context, client EmailSender, to string, subject string, html string, text string) error {
	return client.Send(ctx, to, subject, EmailContent{ContentType: "text/plain", Content: text}, EmailContent{ContentType: "text/html", Content: html})
}

func SendTemplate(ctx op_context.Context, client EmailSender, templatesPath string, templateName string, to string, vals TemplateVals, language ...string) error {

	c := ctx.TraceInMethod("email_client.SendTemplate", logger.Fields{"templates_path": templatesPath, "template": templateName})
	defer ctx.TraceOutMethod()

	// TODO use language path

	subject, err := utils.ReadTemplate(templatesPath, fmt.Sprintf("%v-subject.txt", templateName), nil)
	if err != nil {
		c.SetMessage("failed to read template subject")
		return c.SetError(err)
	}

	text, _ := utils.ReadTemplate(templatesPath, fmt.Sprintf("%v.txt", templateName), vals)
	html, _ := utils.ReadTemplate(templatesPath, fmt.Sprintf("%v.html", templateName), vals)
	if html == "" && text == "" {
		return c.SetErrorStr("empty content")
	}

	if html != "" && text != "" {
		return SendHtmlAndText(ctx, client, to, subject, html, text)
	} else if html != "" {
		return SendHtml(ctx, client, to, subject, html)
	}
	return SendText(ctx, client, to, subject, html)
}
