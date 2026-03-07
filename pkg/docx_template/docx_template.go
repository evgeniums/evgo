package docx_template

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/fatih/structs"
	"github.com/lukasjarosch/go-docx"
)

type DocxTemplateConfig struct {
	LIBREOFFICE_EXECUTABLE string `default:"soffice" validate:"required" vmessage:"Libreoffice executable must be specified"`
	TEMP_DIR               string
}

type DocxTemplate struct {
	DocxTemplateConfig
}

func (d *DocxTemplate) Config() interface{} {
	return &d.DocxTemplateConfig
}

func (d *DocxTemplate) Init(app app_context.Context, configPath ...string) error {

	err := object_config.LoadLogValidateApp(app, d, "docx_template", configPath...)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration for DocxTemplate", err)
	}

	return nil
}

func (d *DocxTemplate) ToDocx(ctx op_context.Context, templateFile string, targetFile string, vars interface{}) error {

	// setup
	c := ctx.TraceInMethod("DocxTemplate.ToDocx")
	defer ctx.TraceOutMethod()

	varMap := structs.Map(vars)

	// open template
	template, err := docx.Open(templateFile)
	if err != nil {
		c.SetMessage("failed to open template")
		return c.SetError(err)
	}

	// render template
	err = template.ReplaceAll(varMap)
	if err != nil {
		c.SetMessage("failed to render template")
		return c.SetError(err)
	}

	// save result
	err = template.WriteToFile(targetFile)
	if err != nil {
		c.SetMessage("failed to save result file")
		return c.SetError(err)
	}

	if !utils.FileExists(targetFile) {
		err = errors.New("failed to find generated docx file")
		return c.SetError(err)
	}

	// done
	return nil
}

func (d *DocxTemplate) ToPdf(ctx op_context.Context, templateFile string, targetFile string, vars interface{}) error {

	// setup
	tempDir := d.TEMP_DIR
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempFileBase := utils.GenerateID()
	tempFileName := filepath.Join(tempDir, fmt.Sprintf("%s.docx", tempFileBase))
	var err error
	c := ctx.TraceInMethod("DocxTemplate.ToPdf")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		} else {
			os.Remove(tempFileName)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()
	c.SetLoggerField("temp_dir", tempDir)
	c.SetLoggerField("template_file", templateFile)
	c.SetLoggerField("temporary_file", tempFileName)
	c.SetLoggerField("target_file", targetFile)

	// create temporary docx
	err = d.ToDocx(ctx, templateFile, tempFileName, vars)
	if err != nil {
		return err
	}

	// convert docx to pdf
	targetDir := filepath.Dir(targetFile)
	cmd := exec.Command(d.LIBREOFFICE_EXECUTABLE, "--headless", "--convert-to", "pdf:writer_pdf_Export", "--outdir", targetDir, tempFileName)
	err = cmd.Run()
	if err != nil {
		c.SetMessage("failed to execute libreoffice")
		return err
	}

	// rename file
	tmpPdfFileName := filepath.Join(targetDir, fmt.Sprintf("%s.pdf", tempFileBase))
	c.SetLoggerField("temporary_file_pdf", tmpPdfFileName)
	err = os.Rename(tmpPdfFileName, targetFile)
	if err != nil {
		c.SetMessage("failed to rename target file")
		return err
	}

	// done
	return nil
}

func New() *DocxTemplate {
	return &DocxTemplate{}
}
