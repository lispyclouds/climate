package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/spf13/cobra"
)

type Handler func(*cobra.Command, []string)

func LoadV3(data []byte) (*libopenapi.DocumentModel[v3.Document], error) {
	document, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, err
	}

	model, errors := document.BuildV3Model()
	for _, err := range errors {
		return nil, fmt.Errorf("Cannot create v3 model: %e", err)
	}

	return model, nil
}

func LoadFileV3(path string) (*libopenapi.DocumentModel[v3.Document], error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return LoadV3(data)
}

func GenCliV3(model libopenapi.DocumentModel[v3.Document], handlers map[string]Handler, rootCmd *cobra.Command) error {
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
				if err := val.Decode(&opts); err != nil {
					return err
				}

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
				// TODO: handle param.In
				// TODO: handle param interpolation
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

			handler, ok := handlers[op.OperationId]
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

	return nil
}
