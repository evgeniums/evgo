package config_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigArgs(t *testing.T) {
	configFile := test_utils.AssetsFilePath(testDir, "main.jsonc")
	app := app_default.New(nil)
	require.NoErrorf(t, app.Init(configFile), "failed to init application context")
	defer app.Close()

	// fmt.Printf("Settings before:\n%s\n", app.Cfg().ToString())

	assert.Equal(t, "value1", app.Cfg().GetString("main_section.parameter1"))

	flistSlice := app.Cfg().GetFloat64Slice("main_section.float_list")
	assert.InDeltaSlice(t, []float64{100.99, 200.01, 300.01}, flistSlice, 0.001)
	require.Equal(t, 3, len(flistSlice))
	assert.InDelta(t, 100.99, flistSlice[0], 0.001)
	assert.InDelta(t, 200.01, flistSlice[1], 0.001)
	assert.InDelta(t, 300.01, flistSlice[2], 0.001)

	args := []string{
		"--main_section.parameter1.string", "arg_value1",
		"--main_section.parameter2.", "arg_value2",
		"--main_section.subsection.int_parameter.int", "1234",
		"--main_section.subsection.float_parameter.float", "100,29",
		"--main_section.subsection.bool_parameter.bool", "true",
		"--main_section.subsection.list.string_list", "one,two,three",
		"--main_section.subsection.ilist.int_list", "1,2,3",
		"--main_section.subsection.flist.float_list", "100.99,200.02,300.03",
	}
	require.NoErrorf(t, config.LoadArgs(app.Cfg(), args), "failed to load args")

	// fmt.Printf("Settings after:\n%s\n", app.Cfg().ToString())

	assert.Equal(t, "arg_value1", app.Cfg().GetString("main_section.parameter1"))
	assert.Equal(t, "arg_value2", app.Cfg().GetString("main_section.parameter2"))
	assert.Equal(t, 1234, app.Cfg().GetInt("main_section.subsection.int_parameter"))
	assert.InDelta(t, 100.29, app.Cfg().GetFloat64("main_section.subsection.float_parameter"), 0.001)
	assert.True(t, app.Cfg().GetBool("main_section.subsection.bool_parameter"))
	assert.Equal(t, []string{"one", "two", "three"}, app.Cfg().GetStringSlice("main_section.subsection.list"))
	assert.Equal(t, []int{1, 2, 3}, app.Cfg().GetIntSlice("main_section.subsection.ilist"))
	assert.InDeltaSlice(t, []float64{100.99, 200.02, 300.03}, app.Cfg().GetFloat64Slice("main_section.subsection.flist"), 0.001)

	app.Cfg().Set("main_section.subsection.flist_manual", []float64{100.99, 200.02, 300.03})
	assert.InDeltaSlice(t, []float64{100.99, 200.02, 300.03}, app.Cfg().GetFloat64Slice("main_section.subsection.flist_manual"), 0.001)
}
