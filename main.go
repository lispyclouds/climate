package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi"
	"github.com/spf13/cobra"
)

type Handler func(*cobra.Command, []string)

func bailIfErr(err error) {
	if err != nil {
		fmt.Printf("Error: %e", err)
		os.Exit(1)
	}
}

// TODO: Should the handler be of a different sig?
func handler(cmd *cobra.Command, args []string) {
}

func main() {
	file, err := os.ReadFile("api.yaml")
	bailIfErr(err)

	document, err := libopenapi.NewDocument(file)
	bailIfErr(err)

	model, errors := document.BuildV3Model()
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Printf("error: %e\n", err)
		}

		bailIfErr(fmt.Errorf("Cannot create v3 model from document: %d errors reported", len(errors)))
	}

	rootCmd := cobra.Command{
		Use:   "bctl",
		Short: "Bob CLI",
		Long:  "This is Bob's CLI mate!",
	}

	handlerMap := map[string]Handler{
		"GetApiSpec":             handler,
		"HealthCheck":            handler,
		"PipelineCreate":         handler,
		"PipelineDelete":         handler,
		"PipelineStart":          handler,
		"PipelineStop":           handler,
		"PipelinePause":          handler,
		"PipelineUnpause":        handler,
		"PipelineLogs":           handler,
		"PipelineStatus":         handler,
		"PipelineArtifactFetch":  handler,
		"PipelineList":           handler,
		"PipelineRuns":           handler,
		"ResourceProviderCreate": handler,
		"ResourceProviderDelete": handler,
		"ResourceProviderList":   handler,
		"ArtifactStoreCreate":    handler,
		"ArtifactStoreDelete":    handler,
		"ArtifactStoreList":      handler,
		"GetError":               handler,
		"GetEvents":              handler,
		"GetMetrics":             handler,
		"CCTray":                 handler,
		"ClusterInfo":            handler,
		"Query":                  handler,
	}

	cmdGroups := make(map[string][]cobra.Command)

	for path, item := range model.Model.Paths.PathItems.FromOldest() {
		// TODO: use the path
		_ = path

		for method, op := range item.GetOperations().FromOldest() {
			// TODO: use the method
			_ = method

			cmd := cobra.Command{}
			groupName := ""
			aliases := []string{}

			for ext, val := range op.Extensions.FromOldest() {
				var opts any
				bailIfErr(val.Decode(&opts))

				switch ext {
				case "x-cli-hidden":
					cmd.Hidden = opts.(bool)
				case "x-cli-aliases":
					for _, alias := range opts.([]any) {
						aliases = append(aliases, alias.(string))
					}
				case "x-cli-group":
					groupName = opts.(string)
					_, ok := cmdGroups[groupName]
					if !ok {
						cmdGroups[groupName] = []cobra.Command{}
					}
				default:
					slog.Warn("TODO: Unhandled extension", "ext", ext, "opts", opts)
				}
			}

			flags := cmd.Flags()
			for _, param := range op.Parameters {
				switch param.Schema.Schema().Type[0] {
				case "string":
					flags.String(param.Name, "", param.Description)
					if req := param.Required; req != nil && *req {
						cmd.MarkFlagRequired(param.Name)
					}
				case "integer":
					flags.Int(param.Name, 0, param.Description)
					if req := param.Required; req != nil && *req {
						cmd.MarkFlagRequired(param.Name)
					}
				default:
					slog.Warn("TODO: Unhandled param", "name", param.Name, "type", param.Schema.Schema().Type[0])
				}
			}

			if body := op.RequestBody; body != nil {
				for mime, kind := range body.Content.FromOldest() {
					switch kind.Schema.Schema().Type[0] {
					case "object":
						paramName := "cli-mate-data" // TODO: hammock on ways to handle the req bodies. Maybe take in a stdin?

						flags.StringP(paramName, "f", "", body.Description)
						if req := body.Required; req != nil && *req {
							cmd.MarkFlagRequired(paramName)
						}
					default:
						slog.Warn("TODO: Unhandled request body", "mime", mime, "type", kind.Schema.Schema().Type[0])
					}
				}
			}

			handler, ok := handlerMap[op.OperationId]
			if !ok {
				slog.Warn("TODO: Unknown op, skipping", "id", op.OperationId)
				continue
			}

			cmd.Use = op.OperationId
			// TODO: hammock on better ways to handle aliases
			if len(aliases) > 0 {
				cmd.Use = aliases[0]
				cmd.Aliases = aliases[1:]
			}
			cmd.Short = op.Description
			cmd.Run = handler

			if groupName != "" {
				cmdGroups[groupName] = append(cmdGroups[groupName], cmd)
			}
		}
	}

	for group, cmds := range cmdGroups {
		groupedCmd := cobra.Command{
			Use:   group,
			Short: fmt.Sprintf("Operations on %s", group),
		}

		for _, cmd := range cmds {
			groupedCmd.AddCommand(&cmd)
		}

		rootCmd.AddCommand(&groupedCmd)
	}

	rootCmd.Execute()
}
