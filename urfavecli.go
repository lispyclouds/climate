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

type HandlerUrfaveCli func(opts *cli.Command, args []string, data HandlerData) error

func addParamsUrfaveCli(cmd *cli.Command, op *v3.Operation, handlerData *HandlerData) {
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
		desc := param.Description
		required := false
		if req := param.Required; req != nil {
			required = *req
		}

		switch t {
		case String:
			flags = append(flags, &cli.StringFlag{
				Name:     name,
				Usage:    desc,
				Required: required,
			})
		case Integer:
			flags = append(flags, &cli.IntFlag{
				Name:     name,
				Usage:    desc,
				Required: required,
			})
		case Number:
			flags = append(flags, &cli.Float64Flag{
				Name:     name,
				Usage:    desc,
				Required: required,
			})
		case Boolean:
			flags = append(flags, &cli.BoolFlag{
				Name:     name,
				Usage:    desc,
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

func addRequestBodyUrfaveCli(cmd *cli.Command, op *v3.Operation, handlerData *HandlerData) error {
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

func interpolatePathUrfaveCli(cmd *cli.Command, h *HandlerData) error {
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
			v := cmd.Int(param.Name)
			value = strconv.FormatInt(int64(v), 10)
		case Number:
			v := cmd.Float64(param.Name)
			value = strconv.FormatFloat(v, 'g', -1, 64)
		case Boolean:
			v := cmd.Bool(param.Name)
			value = strconv.FormatBool(v)
		}

		h.Path = pattern.ReplaceAllString(h.Path, value)
	}

	return nil
}

// Bootstraps a cli.Command with the loaded model and a handler map
func BootstrapV3UrfaveCli(rootCmd *cli.Command, model libopenapi.DocumentModel[v3.Document], handlers map[string]HandlerUrfaveCli) error {
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
			addParamsUrfaveCli(&cmd, op, &hData)
			if err := addRequestBodyUrfaveCli(&cmd, op, &hData); err != nil {
				return err
			}

			handler, ok := handlers[op.OperationId]
			if !ok {
				slog.Warn("No handler defined, skipping", "id", op.OperationId)
				continue
			}

			cmd.Hidden = exts.hidden
			cmd.Aliases = exts.aliases
			cmd.Description = op.Description
			if op.Summary != "" {
				cmd.Description = op.Summary
			}
			cmd.Action = func(_ context.Context, cmd *cli.Command) error {
				if err := interpolatePathUrfaveCli(cmd, &hData); err != nil {
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
			Name:        group,
			Description: fmt.Sprintf("Operations on %s", group),
		}
		groupedCmd.Commands = cmds
		rootCmd.Commands = append(rootCmd.Commands, &groupedCmd)
	}

	return nil
}
