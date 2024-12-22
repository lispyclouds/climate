# climate

[![Go Report Card](https://goreportcard.com/badge/github.com/lispyclouds/climate)](https://goreportcard.com/report/github.com/lispyclouds/climate)
[![CI Status](https://github.com/lispyclouds/climate/workflows/Test/badge.svg)](https://github.com/lispyclouds/climate/actions?query=workflow%3ATest)

Go is a fantastic language to build CLI tooling, specially the ones for interacting with an API server. `<your tool>ctl` anyone?
But if you're tired of building bespoke CLIs everytime or think that the swagger codegen isn't just good enough or don't quite subscribe to the idea of codegen in general (like me!), look no further.

What if you can influence the CLI behaviour from the server? This enables you to bootstrap your [cobra](https://cobra.dev/) CLI tooling from an [OpenAPI](https://swagger.io/specification/) spec.

## Getting started

### Status

Alpha, looking for design/usage feedback!

### Ideally support:
- more of the OpenAPI types and their checks. eg arrays, enums, objects, multi types etc
- type checking request bodies of certain MIME types eg, `application/json`
- better handling of request bodies eg, providing a stdin or a curl like notation for a file `@payload.json` etc.

### Installation

```bash
go get github.com/lispyclouds/climate
```

### Rationale

climate allows the server to influence the CLI behaviour by using OpenAPI's [extensions](https://swagger.io/docs/specification/v3_0/openapi-extensions/). It encourages [spec-first](https://www.atlassian.com/blog/technology/spec-first-api-development) practices thereby keeping both users and maintenance manageable. It does just enough to handle the spec and nothing more.

Overall, the way it works:
- Each operation is converted to a Cobra command
- Each parameter is converted to a flag with its corresponding type
- As of now, request bodies are a flag and treated as a string regardless of MIME type. Name defaults to `climate-data` unless specified via `x-cli-name`. All subject to change
- The provided handlers are attached to each command, grouped and attached to the rootCmd

Influenced by some of the ideas behind [restish](https://rest.sh/) it uses the following extensions as of now:
- `x-cli-aliases`: A list of strings which would be used as the alternate names for an operation
- `x-cli-group`: A string to allow grouping subcommands together. All operations in the same group would become subcommands in that group name
- `x-cli-hidden`: A boolean to hide the operation from the CLI menu. Same behaviour as a cobra command hide: it's present and expects a handler
- `x-cli-ignored`: A boolean to tell climate to omit the operation completely
- `x-cli-name`: A string to specify a different name. Applies to operations and request bodies as of now

### Usage

Given an OpenAPI spec like [api.yaml](/api.yaml)

Load the spec:

```go
model, err := climate.LoadFileV3("api.yaml") // or climate.LoadV3 with []byte
```

Define a cobra root command:

```go
rootCmd := &cobra.Command{
	Use:   "calc",
	Short: "My Calc",
	Long:  "My Calc powered by OpenAPI",
}
```

Define one or more handler functions of the following signature:
```go
func handler(opts *cobra.Command, args []string, data climate.HandlerData) error {
	slog.Info("called!", "data", fmt.Sprintf("%+v", data))
	err := doSomethingUseful(data)

	return err
}
```
#### Handler Data

(Feedback welcome to make this better!)

As of now, each handler is called with the cobra command it was invoked with, the args and an extra `climate.HandlerData`, more info [here](https://pkg.go.dev/github.com/lispyclouds/climate#pkg-types)

This can be used to query the params from the command mostly in a type safe manner:
```go
// to get all the int path params
for _, param := range data.PathParams {
	if param.Type == climate.Integer {
		value, _ := opts.Flags().GetInt(param.Name)
	}
}
```

Define the handlers for the necessary operations. These map to the `operationId` field of each operation:

```go
handlers := map[string]Handler{
	"AddGet":      handler,
	"AddPost":     handler,
	"HealthCheck": handler,
	"GetInfo":     handler,
}
```

Bootstrap the root command:

```go
err := climate.BootstrapV3(rootCmd, *model, handlers)
```

Continue adding more commands and/or execute:

```go
// add more commands not from the spec

rootCmd.Execute()
```

Sample output:

```
$ go run main.go --help
My Calc powered by OpenAPI

Usage:
  calc [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  info        Operations on info
  ops         Operations on ops
  ping        Returns Ok if all is well

Flags:
  -h, --help   help for calc

Use "calc [command] --help" for more information about a command.

$ go run main.go ops --help
Operations on ops

Usage:
  calc ops [command]

Available Commands:
  add-get     Adds two numbers
  add-post    Adds two numbers via POST

Flags:
  -h, --help   help for ops

Use "calc ops [command] --help" for more information about a command.

$ go run main.go ops add-get --help
Adds two numbers

Usage:
  calc ops add-get [flags]

Aliases:
  add-get, ag

Flags:
  -h, --help     help for add-get
      --n1 int   The first number
      --n2 int   The second number

$ go run main.go ops add-get --n1 1 --n2 foo
Error: invalid argument "foo" for "--n2" flag: strconv.ParseInt: parsing "foo": invalid syntax
Usage:
  calc ops add-get [flags]

Aliases:
  add-get, ag

Flags:
  -h, --help     help for add-get
      --n1 int   The first number
      --n2 int   The second number

$ go run main.go ops add-get --n1 1 --n2 2
2024/12/14 12:53:32 INFO called! data="{Method:get Path:/add/{n1}/{n2}}"
```

## License
Copyright © 2024- Rahul De

Distributed under the MIT License. See LICENSE.
