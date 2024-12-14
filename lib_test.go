package climate

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/spf13/cobra"
)

func TestLoadV3(t *testing.T) {
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
		Use:   "bctl",
		Short: "Bob CLI",
		Long:  "This is Bob's CLI mate!",
	}
	handlers := map[string]Handler{
		"ArtifactStoreCreate":    handler,
		"ArtifactStoreDelete":    handler,
		"ArtifactStoreList":      handler,
		"CCTray":                 handler,
		"ClusterInfo":            handler,
		"GetApiSpec":             handler,
		"GetError":               handler,
		"GetEvents":              handler,
		"GetMetrics":             handler,
		"HealthCheck":            handler,
		"PipelineArtifactFetch":  handler,
		"PipelineCreate":         handler,
		"PipelineDelete":         handler,
		"PipelineList":           handler,
		"PipelineLogs":           handler,
		"PipelinePause":          handler,
		"PipelineRuns":           handler,
		"PipelineStart":          handler,
		"PipelineStatus":         handler,
		"PipelineStop":           handler,
		"PipelineUnpause":        handler,
		"Query":                  handler,
		"ResourceProviderCreate": handler,
		"ResourceProviderDelete": handler,
		"ResourceProviderList":   handler,
	}

	err = BootstrapV3(&rootCmd, *model, handlers)
	if err != nil {
		t.Fatalf("BootstrapV3 failed with: %e", err)
	}
}
