package console_tool

import (
	"fmt"
	"syscall"

	"github.com/evgeniums/evgo/pkg/utils"
	"golang.org/x/term"
)

func ReadPassword(message ...string) string {
	msg := utils.OptionalArg("Please, enter new password/secret:", message...)
	fmt.Println(msg)
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(fmt.Sprintf("failed to enter password/secret: %s", err))
	}
	return string(password)
}
