package climate

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestLoadFileV3(t *testing.T) {
	_, err := LoadFileV3("api.yaml")
	if err != nil {
		t.Fatalf("LoadFileV3 failed with: %e", err)
	}
}

func assertCmdTree(t *testing.T, cmd *cobra.Command, assertConf map[string]map[string]any, prefix string) {
	fmt.Println("Checking cmd level " + prefix)

	expected, ok := assertConf[prefix]
	if !ok {
		t.Fatalf("Invalid prefix found: %s", prefix)
	}

	assert.Equal(t, expected["Use"], cmd.Use)
	assert.Equal(t, expected["Short"], cmd.Short)
	assert.Equal(t, expected["Aliases"], cmd.Aliases)

	expectedFlags, ok := expected["Flags"]
	if ok {
		for name, info := range expectedFlags.(map[string]any) {
			typ := OpenAPIType(info.(map[string]OpenAPIType)["Type"])
			var err error

			switch typ {
			case String:
				_, err = cmd.Flags().GetString(name)
			case Integer:
				_, err = cmd.Flags().GetInt(name)
			case Number:
				_, err = cmd.Flags().GetFloat64(name)
			case Boolean:
				_, err = cmd.Flags().GetBool(name)
			}

			assert.NoError(t, err, fmt.Sprintf("Flag: %s Type: %s", name, typ))
		}
	}

	for _, subCmd := range cmd.Commands() {
		assertCmdTree(t, subCmd, assertConf, prefix+"/"+subCmd.Use)
	}
}

func TestBootstrapV3(t *testing.T) {
	model, err := LoadFileV3("api.yaml")
	if err != nil {
		t.Fatalf("LoadFileV3 failed with: %e", err)
	}

	handler := func(opts *cobra.Command, args []string, data HandlerData) {
		// TODO: test handlers when?
	}
	rootCmd := &cobra.Command{
		Use:   "calc",
		Short: "My Calc",
		Long:  "My Calc powered by OpenAPI",
	}
	handlers := map[string]Handler{
		"AddGet":      handler,
		"AddPost":     handler,
		"HealthCheck": handler,
		"GetInfo":     handler,
	}

	err = BootstrapV3(rootCmd, *model, handlers)
	if err != nil {
		t.Fatalf("BootstrapV3 failed with: %e", err)
	}

	var noAlias []string
	assertConf := map[string]map[string]any{
		"calc": {
			"Use":     "calc",
			"Short":   "My Calc",
			"Aliases": noAlias,
		},
		"calc/info": {
			"Use":     "info",
			"Short":   "Operations on info",
			"Aliases": noAlias,
		},
		"calc/info/GetInfo": {
			"Use":     "GetInfo",
			"Short":   "Returns info",
			"Aliases": noAlias,
		},
		"calc/ops": {
			"Use":     "ops",
			"Short":   "Operations on ops",
			"Aliases": noAlias,
		},
		"calc/ops/add-get": {
			"Use":     "add-get",
			"Short":   "Adds two numbers",
			"Aliases": []string{"ag"},
			"Flags": map[string]any{
				"n1": map[string]OpenAPIType{
					"Type": Integer,
				},
				"n2": map[string]OpenAPIType{
					"Type": Integer,
				},
			},
		},
		"calc/ops/add-post": {
			"Use":     "add-post",
			"Short":   "Adds two numbers via POST",
			"Aliases": []string{"ap"},
			"Flags": map[string]any{
				"nmap": map[string]OpenAPIType{
					"Type": String,
				},
			},
		},
		"calc/ping": {
			"Use":     "ping",
			"Short":   "Returns Ok if all is well",
			"Aliases": noAlias,
		},
	}

	assertCmdTree(t, rootCmd, assertConf, rootCmd.Use)
}
