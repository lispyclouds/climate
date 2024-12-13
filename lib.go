package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Handler func(*cobra.Command, []string)

type extensions struct {
	hidden  bool
	aliases []string
	group   string
}

func parseExtensions(exts *orderedmap.Map[string, *yaml.Node]) (*extensions, error) {
	ex := extensions{}
	aliases := []string{}

	for ext, val := range exts.FromOldest() {
		var opts any
		if err := val.Decode(&opts); err != nil {
			return nil, err
		}

		switch ext {
		case "x-cli-hidden":
			ex.hidden = opts.(bool)
		case "x-cli-aliases":
			for _, alias := range opts.([]any) {
				aliases = append(aliases, alias.(string))
			}
			ex.aliases = aliases
		case "x-cli-group":
			ex.group = opts.(string)
		default:
			slog.Warn("TODO: Unhandled extension", "ext", ext, "opts", opts)
		}
	}

	return &ex, nil
}

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
			exts, err := parseExtensions(op.Extensions)
			if err != nil {
				return err
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

			cmd.Hidden = exts.hidden
			cmd.Use = op.OperationId
			cmd.Short = op.Description
			cmd.Run = handler

			// TODO: hammock on better ways to handle aliases
			if aliases := exts.aliases; len(exts.aliases) > 0 {
				cmd.Use = aliases[0]
				cmd.Aliases = aliases[1:]
			}

			// TODO: what if there is no group?
			if g := exts.group; g != "" {
				_, ok := cmdGroups[g]
				if !ok {
					cmdGroups[g] = []cobra.Command{}
				}
				cmdGroups[g] = append(cmdGroups[g], cmd)
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
