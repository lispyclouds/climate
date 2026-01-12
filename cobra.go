// Copyright 2025 Rahul De
// SPDX-License-Identifier: MIT

// climate allows the server to influence the CLI behaviour by using OpenAPI's extensions.
// It encourages spec-first practices thereby keeping both users and maintenance manageable.
// It does just enough to handle the spec and nothing more.

package climate

import (
	"fmt"
	"log/slog"
	"regexp"
	"strconv"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/spf13/cobra"
)

type Handler func(opts *cobra.Command, args []string, data HandlerData) error
type HandlerCobra Handler

func addParams(cmd *cobra.Command, op *v3.Operation, handlerData *HandlerData) {
	var (
		queryParams  []ParamMeta
		pathParams   []ParamMeta
		headerParams []ParamMeta
		cookieParams []ParamMeta
	)
	flags := cmd.Flags()

	for _, param := range op.Parameters {
		t := getParamType(param, op)

		switch t {
		case String:
			flags.String(param.Name, "", param.Description)
		case Integer:
			flags.Int(param.Name, 0, param.Description)
		case Number:
			flags.Float64(param.Name, 0.0, param.Description)
		case Boolean:
			flags.Bool(param.Name, false, param.Description)
		default:
			// TODO: array, object
			slog.Warn("TODO: Unhandled param", "name", param.Name, "type", param.Schema.Schema().Type[0])
			continue
		}

		// TODO: Extract commom
		meta := ParamMeta{Name: param.Name, Type: t}
		switch param.In {
		case "path":
			pathParams = append(pathParams, meta)
		case "query":
			queryParams = append(queryParams, meta)
		case "header":
			headerParams = append(headerParams, meta)
		case "cookie":
			cookieParams = append(cookieParams, meta)
		}

		if req := param.Required; req != nil && *req {
			cmd.MarkFlagRequired(param.Name)
		}
	}

	handlerData.QueryParams = queryParams
	handlerData.PathParams = pathParams
	handlerData.HeaderParams = headerParams
	handlerData.CookieParams = cookieParams
}

func addRequestBodyCobra(cmd *cobra.Command, op *v3.Operation, handlerData *HandlerData) error {
	name, desc, required, err := makeRequestBody(op, handlerData)
	if err != nil {
		return err
	}

	cmd.Flags().String(name, "", desc)

	if required {
		cmd.MarkFlagRequired(name)
	}

	return nil
}

func interpolatePathCobra(cmd *cobra.Command, h *HandlerData) error {
	// TODO: Extract commom
	flags := cmd.Flags()

	for _, param := range h.PathParams {
		pattern, err := regexp.Compile(fmt.Sprintf("({%s})+", param.Name))
		if err != nil {
			return err
		}

		var value string

		switch param.Type {
		case String:
			value, _ = flags.GetString(param.Name)
		case Integer:
			v, _ := flags.GetInt(param.Name)
			value = strconv.FormatInt(int64(v), 10)
		case Number:
			v, _ := flags.GetFloat64(param.Name)
			value = strconv.FormatFloat(v, 'g', -1, 64)
		case Boolean:
			v, _ := flags.GetBool(param.Name)
			value = strconv.FormatBool(v)
		}

		h.Path = pattern.ReplaceAllString(h.Path, value)
	}

	return nil
}

// Bootstraps a cobra.Command with the loaded model and a handler map
func BootstrapV3Cobra(rootCmd *cobra.Command, model libopenapi.DocumentModel[v3.Document], handlers map[string]Handler) error {
	cmdGroups := make(map[string][]cobra.Command)

	for path, item := range model.Model.Paths.PathItems.FromOldest() {
		for method, op := range item.GetOperations().FromOldest() {
			cmd := cobra.Command{}
			exts, err := parseExtensions(op.Extensions)
			if err != nil {
				return err
			}

			if exts.ignored {
				continue
			}

			hData := HandlerData{Method: method, Path: path}
			addParams(&cmd, op, &hData)
			if err := addRequestBodyCobra(&cmd, op, &hData); err != nil {
				return err
			}

			handler, ok := handlers[op.OperationId]
			if !ok {
				slog.Warn("No handler defined, skipping", "id", op.OperationId)
				continue
			}

			cmd.Hidden = exts.hidden
			cmd.Aliases = exts.aliases
			cmd.Short = op.Description
			if op.Summary != "" {
				cmd.Short = op.Summary
			}
			cmd.RunE = func(opts *cobra.Command, args []string) error {
				if err := interpolatePathCobra(&cmd, &hData); err != nil {
					return err
				}

				return handler(opts, args, hData)
			}

			cmd.Use = op.OperationId // default
			if altName := exts.name; altName != "" {
				cmd.Use = altName
			}

			if g := exts.group; g != "" {
				_, ok := cmdGroups[g]
				if !ok {
					cmdGroups[g] = []cobra.Command{}
				}
				cmdGroups[g] = append(cmdGroups[g], cmd)
			} else {
				rootCmd.AddCommand(&cmd)
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

// Bootstraps a cobra.Command with the loaded model and a handler map
//
// Deprecated: Will be kept for backwards compatibility.
// Use BootstrapV3Cobra instead.
func BootstrapV3(rootCmd *cobra.Command, model libopenapi.DocumentModel[v3.Document], handlers map[string]Handler) error {
	return BootstrapV3Cobra(rootCmd, model, handlers)
}
