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

func TestBootstrapV3(t *testing.T) {
	model, err := LoadFileV3("api.yaml")
	if err != nil {
		t.Fatalf("LoadFileV3 failed with: %e", err)
	}

	handler := func(opts *cobra.Command, args []string, data HandlerData) {
		slog.Info("called!", "data", fmt.Sprintf("%+v", data))
	}
	rootCmd := cobra.Command{
		Use:   "calc",
		Short: "My Calc",
		Long:  "My Calc powered by OpenAPI",
	}
	handlers := map[string]Handler{
		"AddGet":      handler,
		"AddPost":     handler,
		"HealthCheck": handler,
		"GetMeta":     handler,
	}

	err = BootstrapV3(&rootCmd, *model, handlers)
	if err != nil {
		t.Fatalf("BootstrapV3 failed with: %e", err)
	}

	names := []string{}
	for _, cmd := range rootCmd.Commands() {
		names = append(names, cmd.Name())
	}
	assert.Contains(t, names, "ops")
	assert.Contains(t, names, "ping")

	names = []string{}
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "ops" {
			for _, subCmd := range cmd.Commands() {
				names = append(names, subCmd.Name())
			}
		}
	}
	assert.Contains(t, names, "add-get")
	assert.Contains(t, names, "add-post")
}
