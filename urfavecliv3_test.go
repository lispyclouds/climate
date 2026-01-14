package climate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestInterpolatePathUrfaveCliV3(t *testing.T) {
	hData := HandlerData{
		Method: "get",
		Path:   "/path/{foo}/to/{bar}/with/{baz}/and/{quxx}/together/{foo}",
		PathParams: []ParamMeta{
			{Name: "foo", Type: String},
			{Name: "bar", Type: Integer},
			{Name: "baz", Type: Number},
			{Name: "quxx", Type: Boolean},
		},
	}
	cmd := cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "foo",
				Usage: "foo usage",
				Value: "yes",
			},
			&cli.IntFlag{
				Name:  "bar",
				Usage: "bar usage",
				Value: 420,
			},
			&cli.Float64Flag{
				Name:  "baz",
				Usage: "baz usage",
				Value: 420.69,
			},
			&cli.BoolFlag{
				Name:  "quxx",
				Usage: "quxx usage",
				Value: false,
			},
		},
	}

	err := interpolatePathUrfaveCliV3(&cmd, &hData)
	assert.NoError(t, err)

	assert.Equal(t, hData.Path, "/path/yes/to/420/with/420.69/and/false/together/yes")
}

func assertCmdTreeUrfaveCliV3(t *testing.T, cmd *cli.Command, expected *cli.Command) {
	t.Logf("Checking command %s", cmd.Name)

	assert.Equal(t, expected.Name, cmd.Name)
	assert.Equal(t, expected.Usage, cmd.Usage)
	assert.Equal(t, expected.Aliases, cmd.Aliases)

	cmd.InvalidFlagAccessHandler = func(_ context.Context, cmd *cli.Command, name string) {
		t.Logf("Invalid flag accessed %s in command %v", name, cmd)
		t.FailNow()
	}

	for _, flag := range expected.Flags {
		switch flag.(type) {
		case *cli.StringFlag:
			cmd.String(flag.Names()[0])
		case *cli.IntFlag:
			cmd.Int(flag.Names()[0])
		case *cli.Float64Flag:
			cmd.Float64(flag.Names()[0])
		case *cli.BoolFlag:
			cmd.Bool(flag.Names()[0])
		}
	}

	for _, subCmd := range cmd.Commands {
		assertCmdTreeUrfaveCliV3(t, subCmd, expected.Command(subCmd.Name))
	}
}

func TestBootstrapV3UrfaveCliV3(t *testing.T) {
	model, err := LoadFileV3("api.yaml")
	assert.NoError(t, err)

	handler := func(opts *cli.Command, args []string, data HandlerData) error {
		assert.Equal(t, data.PathParams, []ParamMeta{{Name: "p1", Type: Integer}})
		assert.Equal(t, data.QueryParams, []ParamMeta{{Name: "p2", Type: String}})
		assert.Equal(t, data.HeaderParams, []ParamMeta{{Name: "p3", Type: Number}})
		assert.Equal(t, data.CookieParams, []ParamMeta{{Name: "p4", Type: Boolean}})
		assert.Equal(t, data.RequestBodyParam, &ParamMeta{Name: "req-body", Type: String})

		return nil
	}
	rootCmd := &cli.Command{
		Name:  "calc",
		Usage: "My Calc",
	}
	handlers := map[string]HandlerUrfaveCliV3{
		"AddGet":      handler,
		"AddPost":     handler,
		"HealthCheck": handler,
		"GetInfo":     handler,
	}

	err = BootstrapV3UrfaveCliV3(rootCmd, *model, handlers)
	assert.NoError(t, err)

	var noAlias []string
	expectedCmd := &cli.Command{
		Name:    "calc",
		Usage:   "My Calc",
		Aliases: noAlias,
		Commands: []*cli.Command{
			{
				Name:    "info",
				Usage:   "Operations on info",
				Aliases: noAlias,
				Commands: []*cli.Command{
					{
						Name:    "GetInfo",
						Usage:   "Returns info",
						Aliases: noAlias,
					},
				},
			},
			{
				Name:    "ops",
				Usage:   "Operations on ops",
				Aliases: noAlias,
				Commands: []*cli.Command{
					{
						Name:    "add-get",
						Usage:   "Adds two numbers",
						Aliases: []string{"ag"},
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name: "n1",
							},
							&cli.IntFlag{
								Name: "n2",
							},
						},
					},
					{
						Name:    "add-post",
						Usage:   "Adds two numbers via POST",
						Aliases: []string{"ap"},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name: "nmap",
							},
						},
					},
				},
			},
			{
				Name:    "ping",
				Usage:   "Returns Ok if all is well",
				Aliases: noAlias,
			},
		},
	}

	assertCmdTreeUrfaveCliV3(t, rootCmd, expectedCmd)

	assert.NoError(t, rootCmd.Run(
		context.Background(),
		[]string{
			"calc",
			"info",
			"GetInfo",
			"--p1",
			"420",
			"--p2",
			"yes",
			"--p3",
			"420.69",
			"--p4",
			"true",
			"--req-body",
			"the string body",
		},
	))
}
