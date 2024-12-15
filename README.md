# climate

[![Go Report Card](https://goreportcard.com/badge/github.com/lispyclouds/climate)](https://goreportcard.com/report/github.com/lispyclouds/climate)
[![CI Status](https://github.com/lispyclouds/climate/workflows/Test/badge.svg)](https://github.com/lispyclouds/climate/actions?query=workflow%3ATest)

Go is a fantastic language to build CLI tooling, specially the ones for interacting with an API server. `<your tool>ctl` anyone?
But if you're tired of building bespoke CLIs everytime or think that the swagger codegen isn't just good enough, look no further.

What if you can influence the CLI behaviour from the server? This enables you to bootstrap your [cobra](https://cobra.dev/) CLI tooling from an [OpenAPI](https://swagger.io/specification/) spec.

## Getting started

### Status

Experimental, in dev flux and looking for design/usage feedback!

### TODO:
- Interpolate the HTTP paths when sending to the handler with the path params
- Support more of the OpenAPI types and their checks. eg arrays, enums, objects, multi types etc
- Much better unit tests touching just the public fns and assert shape
- Type checking request bodies

### Installation

```bash
go get github.com/lispyclouds/climate
```

### How it works and usage

climate allows the server to influence the CLI behaviour by using OpenAPI's [extensions](https://swagger.io/docs/specification/v3_0/openapi-extensions/). It encourages [spec-first](https://www.atlassian.com/blog/technology/spec-first-api-development) practices thereby keeping both users and maintenance manageable. It does just enough to handle the spec and nothing more.

Overall, the way it works:
- Each operation is converted to a Cobra command
- Each parameter is converted to a flag with its corresponding type
- Request bodies are a flag as of now, subject to change
- The provided handlers are attached to each command, grouped and attached to the rootCmd

Influenced by some of the ideas behind [restish](https://rest.sh/) it uses the following extensions as of now:
- `x-cli-aliases`: A list of strings which would be used as the alternate names for:
  - Operations: If set, will prefer the first of the list otherwise the `operationId`. Will use the rest as cobra aliases
  - Request Body: Same preference as above but would a default of `climate-data` as the name of the param if not set
- `x-cli-group`: A string to allow grouping subcommands together. All operations in the same group would become subcommands in that group name
- `x-cli-hidden`: A boolean to hide the operation from the CLI menu. Same behaviour as a cobra command hide: it's present and expects a handler
- `x-cli-ignored`: A boolean to tell climate to omit the operation completely

Given an OpenAPI spec in `api.yaml`:

```yaml
openapi: "3.0.0"

info:
  title: My calculator
  version: "0.1.0"
  description: My awesome calc!

paths:
  "/add/{n1}/{n2}":
    get:
      operationId: AddGet
      summary: Adds two numbers
      x-cli-group: ops
      x-cli-aliases:
        - add-get
        - ag

      parameters:
        - name: n1
          required: true
          in: path
          description: The first number
          schema:
            type: integer
        - name: n2
          required: true
          in: path
          description: The second number
          schema:
            type: integer
    post:
      operationId: AddPost
      summary: Adds two numbers via POST
      x-cli-group: ops
      x-cli-aliases:
        - add-post
        - ap

      requestBody:
        description: The numbers map
        required: true
        x-cli-aliases:
          - nmap
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NumbersMap"
  "/health":
    get:
      operationId: HealthCheck
      summary: Returns Ok if all is well
      x-cli-aliases:
        - ping
  "/meta":
    get:
      operationId: GetMeta
      summary: Returns meta
      x-cli-ignored: true
  "/info":
    get:
      operationId: GetInfo
      summary: Returns info
      x-cli-group: info

components:
  schemas:
    NumbersMap:
      type: object
      required:
        - n1
        - n2
      properties:
        n1:
          type: integer
          description: The first number
        n2:
          type: integer
          description: The second number
```

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
func handler(opts *cobra.Command, args []string, data climate.HandlerData) {
    // do something more useful
    slog.Info("called!", "data", fmt.Sprintf("%+v", data))
}
```
#### Handler Data

(Feedback welcome to make this better!)

As of now, each handler is called with the cobra command it was invoked with, the args and an extra `climate.HandlerData`

The handler data is of the following structure:
```go
type ParamMeta struct {
	Name string
	Type string // Same as the type name in OpenAPI
}

type HandlerData struct {
	Method           string      // the HTTP method
	Path             string      // the parameterised path
	PathParams       []ParamMeta // List of path params
	QueryParams      []ParamMeta // List of query params
	HeaderParams     []ParamMeta // List of header params
	CookieParams     []ParamMeta // List of cookie params
	RequestBodyParam ParamMeta   // The request body
}
```

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
