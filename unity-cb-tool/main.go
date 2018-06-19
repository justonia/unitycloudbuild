package main

import (
	"log"
	"os"

	cb "github.com/justonia/unitycloudbuild"
	"github.com/urfave/cli"
)

const Version string = "0.1.0"

func main() {
	var apiKey string

	app := cli.NewApp()
	app.Name = "unity-cb-tool"
	app.Version = Version
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "api-key",
			Usage:       "Unity API key",
			EnvVar:      "UNITY_API_KEY",
			Destination: &apiKey,
		},
		cli.StringFlag{
			Name:   "org-id",
			Usage:  "Unity Organization ID",
			EnvVar: "UNITY_ORG_ID",
		},
		cli.StringFlag{
			Name:   "project-id",
			Usage:  "Unity Project ID",
			EnvVar: "UNITY_PROJECT_ID",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "If true, output responses in JSON",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "If true, output detailed status messages to log",
		},
	}
	app.Commands = []cli.Command{
		{
			Name: "builds",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List builds",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "target-id,t",
							Usage: "Specific target ID or _all for all targets",
							Value: "_all",
						},
						cli.StringFlag{
							Name:  "filter-status",
							Usage: "(queued, sentToBuilder, started, restarted, success, failure, canceled, unknown)",
						},
						cli.StringFlag{
							Name:  "filter-platform",
							Usage: "(ios, android, webgl, osx, win, win64, linux)",
						},
						cli.Int64Flag{
							Name:  "limit,l",
							Usage: "If >0 show only the specified number of builds",
						},
					},
					Action: func(c *cli.Context) error {
						_, err := cb.Builds_List(
							buildContext(c),
							c.String("target-id"), c.String("filter-status"), c.String("filter-platform"), c.Int64("limit"))
						return err
					},
				},
				{
					Name:  "latest",
					Usage: "List latest builds for every build target",
					Flags: []cli.Flag{},
					Action: func(c *cli.Context) error {
						_, err := cb.Builds_Latest(buildContext(c))
						return err
					},
				},
				{
					Name:  "cancel",
					Usage: "Cancel a build for a build target, or if --all is specified cancel all builds",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "all",
							Usage: "If true, cancel all builds",
						},
						cli.StringFlag{
							Name:  "target-id,t",
							Usage: "Build target ID",
							Value: "",
						},
						cli.Int64Flag{
							Name:  "build,b",
							Usage: "Build number for build target",
							Value: -1,
						},
					},
					Action: func(c *cli.Context) error {
						var err error

						if c.Bool("all") {
							err = cb.Builds_CancelAll(buildContext(c), c.String("target-id"))
						} else {
							if len(c.String("target-id")) == 0 {
								log.Fatal("missing target-id")
							}
							err = cb.Builds_Cancel(buildContext(c), c.String("target-id"), c.Int64("build"))
						}
						return err
					},
				},
			},
		},
		{
			Name: "targets",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "List all build targets",
					Flags: []cli.Flag{},
					Action: func(c *cli.Context) error {
						_, err := cb.Targets_List(buildContext(c))
						return err
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func buildContext(c *cli.Context) *cb.CloudBuildContext {
	apiKey := c.GlobalString("api-key")
	if len(apiKey) == 0 {
		log.Fatal("Missing api-key")
	}

	orgId := c.GlobalString("org-id")
	if len(orgId) == 0 {
		log.Fatal("Missing org-id")
	}

	projectId := c.GlobalString("project-id")
	if len(projectId) == 0 {
		log.Fatal("Missing project-id")
	}

	outputFormat := cb.OutputFormat_Human
	if c.GlobalBool("json") {
		outputFormat = cb.OutputFormat_JSON
	}

	return &cb.CloudBuildContext{
		OrgId:        orgId,
		ProjectId:    projectId,
		ApiKey:       apiKey,
		OutputFormat: outputFormat,
		Verbose:      c.GlobalBool("verbose"),
	}
}
