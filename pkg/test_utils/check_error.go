package test_utils

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func CheckGenericError(t *testing.T, err error, expectedCode string, expectedMessage ...string) {
	assert.Error(t, err)
	gErr, ok := err.(generic_error.Error)
	require.True(t, ok)
	assert.Equal(t, expectedCode, gErr.Code())
	if len(expectedMessage) != 0 {
		assert.Equal(t, expectedMessage[0], gErr.Message())
	}
}

// CheckGenericErrorTaxonomy asserts the family/disposition half of the error contract in
// whitemdesktop/docs/error-contract.md, on top of CheckGenericError's code/message check - i.e.
// that the code was registered with the family it claims to belong to and derives (or was
// explicitly overridden to) the expected disposition, not just that its bare code string matches.
func CheckGenericErrorTaxonomy(t *testing.T, err error, expectedCode string, expectedFamily string, expectedDisposition generic_error.Disposition) {
	CheckGenericError(t, err, expectedCode)
	gErr := err.(generic_error.Error)
	assert.Equal(t, expectedFamily, gErr.Family())
	assert.Equal(t, expectedDisposition, gErr.Disposition())
}

func DumpError(t *testing.T, err error, message ...string) {
	if err == nil {
		t.Logf("%s: no error", utils.OptionalArg("Dump error", message...))
		return
	}
	gErr, ok := err.(generic_error.Error)
	if ok {
		DumpObject(t, gErr, message...)
	} else {
		t.Logf("%s: %s", utils.OptionalArg("Error", message...), err)
	}
}

func ObjectEqual(t *testing.T, left common.Object, right common.Object) {
	if left.GetCreatedAt().Equal(right.GetCreatedAt()) {
		right.SetCreatedAt(left.GetCreatedAt())
	}
	if left.GetUpdatedAt().Equal(right.GetUpdatedAt()) {
		right.SetUpdatedAt(left.GetUpdatedAt())
	}
	assert.Equal(t, left, right)
}

func NoError(t *testing.T, sctx context.Context, err error) {
	if err == nil {
		return
	}
	ctx := op_context.OpContext[op_context.Context](sctx)
	if ctx != nil {
		ctx.Close(sctx)
		gErr := ctx.GenericError()
		if gErr != nil {
			DumpObject(t, gErr, "Generic error")
		}
	}
	require.NoError(t, err)
}

func NoErrorApp(t *testing.T, app app_context.Context, err error) {
	if err != nil && app != nil {
		app.Logger().CheckFatalStack(app.Logger())
	}
	require.NoError(t, err)
}

func DumpObject(t *testing.T, obj interface{}, message ...string) {
	result, err := json.MarshalIndent(obj, " ", " ")
	require.NoError(t, err)
	msg := utils.OptionalString("", message...)
	if msg != "" {
		t.Logf("%s:\n%s\n", msg, string(result))
	} else {
		t.Logf("\n%s\n", string(result))
	}
}
