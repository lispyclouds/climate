// Copyright 2025 Rahul De
// SPDX-License-Identifier: MIT

// climate allows the server to influence the CLI behaviour by using OpenAPI's extensions.
// It encourages spec-first practices thereby keeping both users and maintenance manageable.
// It does just enough to handle the spec and nothing more.

package climate

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/urfave/cli/v3"
)

type HandlerUrfaveCliV3 func(opts *cli.Command, args []string, data HandlerData) error

func addParamsUrfaveCliV3(cmd *cli.Command, op *v3.Operation, handlerData *HandlerData) {
	var (
		queryParams  []ParamMeta
		pathParams   []ParamMeta
		headerParams []ParamMeta
		cookieParams []ParamMeta
	)
	flags := []cli.Flag{}

	for _, param := range op.Parameters {
		t := getParamType(param, op)
		name := param.Name
		usage := param.Description
		required := false
		if req := param.Required; req != nil {
			required = *req
		}

		switch t {
		case String:
			flags = append(flags, &cli.StringFlag{
				Name:     name,
				Usage:    usage,
				Required: required,
			})
		case Integer:
			flags = append(flags, &cli.IntFlag{
				Name:     name,
				Usage:    usage,
				Required: required,
			})
		case Number:
			flags = append(flags, &cli.Float64Flag{
				Name:     name,
				Usage:    usage,
				Required: required,
			})
		case Boolean:
			flags = append(flags, &cli.BoolFlag{
				Name:     name,
				Usage:    usage,
				Required: required,
			})
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
	}

	cmd.Flags = flags
	handlerData.QueryParams = queryParams
	handlerData.PathParams = pathParams
	handlerData.HeaderParams = headerParams
	handlerData.CookieParams = cookieParams
}

func addRequestBodyUrfaveCliV3(cmd *cli.Command, op *v3.Operation, handlerData *HandlerData) error {
	name, desc, required, err := makeRequestBody(op, handlerData)
	if err != nil {
		return err
	}

	cmd.Flags = append(cmd.Flags, &cli.StringFlag{
		Name:     name,
		Usage:    desc,
		Required: required,
	})

	return nil
}

func interpolatePathUrfaveCliV3(cmd *cli.Command, h *HandlerData) error {
	// TODO: Extract commom
	for _, param := range h.PathParams {
		pattern, err := regexp.Compile(fmt.Sprintf("({%s})+", param.Name))
		if err != nil {
			return err
		}

		var value string

		switch param.Type {
		case String:
			value = cmd.String(param.Name)
		case Integer:
			value = strconv.FormatInt(int64(cmd.Int(param.Name)), 10)
		case Number:
			value = strconv.FormatFloat(cmd.Float64(param.Name), 'g', -1, 64)
		case Boolean:
			value = strconv.FormatBool(cmd.Bool(param.Name))
		}

		h.Path = pattern.ReplaceAllString(h.Path, value)
	}

	return nil
}

// Bootstraps a cli.Command with the loaded model and a handler map
func BootstrapV3UrfaveCliV3(rootCmd *cli.Command, model libopenapi.DocumentModel[v3.Document], handlers map[string]HandlerUrfaveCliV3) error {
	cmdGroups := make(map[string][]*cli.Command)

	for path, item := range model.Model.Paths.PathItems.FromOldest() {
		for method, op := range item.GetOperations().FromOldest() {
			cmd := cli.Command{}
			exts, err := parseExtensions(op.Extensions)
			if err != nil {
				return err
			}

			if exts.ignored {
				continue
			}

			hData := HandlerData{Method: method, Path: path}
			addParamsUrfaveCliV3(&cmd, op, &hData)
			if err := addRequestBodyUrfaveCliV3(&cmd, op, &hData); err != nil {
				return err
			}

			handler, ok := handlers[op.OperationId]
			if !ok {
				slog.Warn("No handler defined, skipping", "id", op.OperationId)
				continue
			}

			cmd.Hidden = exts.hidden
			cmd.Aliases = exts.aliases
			cmd.Usage = op.Description
			if op.Summary != "" {
				cmd.Usage = op.Summary
			}
			cmd.Action = func(_ context.Context, cmd *cli.Command) error {
				if err := interpolatePathUrfaveCliV3(cmd, &hData); err != nil {
					return err
				}

				return handler(cmd, cmd.Args().Slice(), hData)
			}

			cmd.Name = op.OperationId // default
			if altName := exts.name; altName != "" {
				cmd.Name = altName
			}

			if g := exts.group; g != "" {
				_, ok := cmdGroups[g]
				if !ok {
					cmdGroups[g] = []*cli.Command{}
				}
				cmdGroups[g] = append(cmdGroups[g], &cmd)
			} else {
				rootCmd.Commands = append(rootCmd.Commands, &cmd)
			}
		}
	}

	for group, cmds := range cmdGroups {
		groupedCmd := cli.Command{
			Name:  group,
			Usage: fmt.Sprintf("Operations on %s", group),
		}
		groupedCmd.Commands = cmds
		rootCmd.Commands = append(rootCmd.Commands, &groupedCmd)
	}

	return nil
}
