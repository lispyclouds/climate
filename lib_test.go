package climate

import (
	"fmt"
	"log/slog"
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

func checkCmdNames(t *testing.T, cmd *cobra.Command, names ...string) {
	actual := []string{}
	for _, cmd := range cmd.Commands() {
		actual = append(actual, cmd.Name())
	}

	assert.Subset(t, names, actual)
}

func TestBootstrapV3(t *testing.T) {
	model, err := LoadFileV3("api.yaml")
	if err != nil {
		t.Fatalf("LoadFileV3 failed with: %e", err)
	}

	handler := func(opts *cobra.Command, args []string, data HandlerData) {
		slog.Info("called!", "data", fmt.Sprintf("%+v", data))
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

	checkCmdNames(t, rootCmd, "ops", "ping", "info")

	var subCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "ops" {
			subCmd = cmd
			break
		}
	}

	checkCmdNames(t, subCmd, "add-get", "add-post")
}
