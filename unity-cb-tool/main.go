package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	cb "github.com/justonia/unitycloudbuild"
	"github.com/urfave/cli"
	yaml "gopkg.in/yaml.v2"
)

const Version string = "0.2.3"

func main() {
	var apiKey string

	app := cli.NewApp()
	app.Name = "unity-cb-tool"
	app.Version = Version
	app.Usage = "A tool to interact with Unity Cloud Build"
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
					Name:  "status",
					Usage: "Retrieve status of a build",
					Flags: []cli.Flag{
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
						if len(c.String("target-id")) == 0 {
							log.Fatal("missing target-id")
						}

						if c.Int64("build") < 0 {
							log.Fatal("missing build number")
						}

						_, err := cb.Builds_Status(buildContext(c), c.String("target-id"), c.Int64("build"))
						return err
					},
				},
				{
					Name:  "latest",
					Usage: "List latest builds for every build target",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "success",
							Usage: "If true, only show latest successful build",
						},
						cli.BoolFlag{
							Name:  "only-enabled",
							Usage: "If true, only show builds from enabled targets",
						},
					},
					Action: func(c *cli.Context) error {
						_, err := cb.Builds_Latest(buildContext(c), c.Bool("success"), c.Bool("only-enabled"))
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
				{
					Name:  "start",
					Usage: "Start a build for a build target, or if --all is specified start builds for all enabled targets",
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "all",
							Usage: "If true, start builds on all enabled targets",
						},
						cli.BoolFlag{
							Name:  "clean",
							Usage: "Force a clean build.",
						},
						cli.StringFlag{
							Name:  "target-id,t",
							Usage: "Build target ID",
							Value: "",
						},
					},
					Action: func(c *cli.Context) error {
						var err error

						if c.Bool("all") {
							_, err = cb.Builds_StartAll(buildContext(c), c.Bool("clean"))
						} else {
							if len(c.String("target-id")) == 0 {
								log.Fatal("missing target-id")
							}
							_, err = cb.Builds_Start(buildContext(c), c.String("target-id"), c.Bool("clean"))
						}
						return err
					},
				},
				{
					Name:  "download",
					Usage: "Download builds",
					Flags: []cli.Flag{
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
						cli.BoolFlag{
							Name:  "latest",
							Usage: "If true, download the latest successful build",
						},
						cli.StringFlag{
							Name:  "output,o",
							Usage: "If set, the build is written to this directory instead of the current directory",
						},
						cli.BoolFlag{
							Name:  "unzip",
							Usage: "If true, unzip the contents of the build to the output directory. Only works with .zip builds (e.g. not .apk)",
						},
					},
					Action: func(c *cli.Context) error {
						if len(c.String("target-id")) == 0 {
							log.Fatal("missing target-id")
						}

						err := cb.Builds_Download(
							buildContext(c),
							c.String("target-id"), c.Int64("build"), c.Bool("latest"), c.String("output"), c.Bool("unzip"))
						return err
					},
				},
				{
					Name:  "wait-for-complete",
					Usage: "Wait for in-progress build(s) to finish",
					Flags: []cli.Flag{
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
						cli.BoolFlag{
							Name:  "all",
							Usage: "If true, wait for all active builds for all enabled targets",
						},
						cli.BoolFlag{
							Name:  "abort-on-fail",
							Usage: "If true, and --all is specified, exit as soon as one build fails or is canceled.",
						},
					},
					Action: func(c *cli.Context) error {
						if !c.Bool("all") {
							if len(c.String("target-id")) == 0 {
								log.Fatal("missing target-id")
							}
						}

						err := cb.Builds_WaitForComplete(
							buildContext(c),
							c.String("target-id"), c.Int64("build"), c.Bool("all"), c.Bool("abort-on-fail"))
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
		{
			Name: "git",
			Subcommands: []cli.Command{
				{
					Name:  "head",
					Usage: "Output current revision and commit message for HEAD",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "repo-path,p",
							Usage: "If set, search for Git repo there instead of current working directory",
						},
					},
					Action: func(c *cli.Context) error {
						_, err := cb.Git_Head(buildContext(c), c.String("repo-path"))
						return err
					},
				},
				{
					Name:  "build-matches-head",
					Usage: "Determine if the build(s) match the current HEAD revision",
					Flags: []cli.Flag{
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
						cli.BoolFlag{
							Name:  "all",
							Usage: "If true, check if all enabled targets match",
						},
						cli.StringFlag{
							Name:  "repo-path,p",
							Usage: "If set, search for Git repo there instead of current working directory",
						},
					},
					Action: func(c *cli.Context) error {
						if !c.Bool("all") && len(c.String("target-id")) == 0 {
							log.Fatal("missing target-id")
						}

						matches, err := cb.Git_BuildsMatchHead(buildContext(c), c.String("repo-path"), c.String("target-id"), c.Int64("build"), c.Bool("all"))
						if err != nil {
							return err
						}
						if !matches {
							return fmt.Errorf("Build(s) do not match.")
						}
						return nil
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

	outputFormat := cb.OutputFormat_Human
	if c.GlobalBool("json") {
		outputFormat = cb.OutputFormat_JSON
	}

	context := &cb.CloudBuildContext{
		OrgId:        c.GlobalString("org-id"),
		ProjectId:    c.GlobalString("project-id"),
		ApiKey:       apiKey,
		OutputFormat: outputFormat,
		Verbose:      c.GlobalBool("verbose"),
	}

	tryFillFromProjectSettings(context)

	if len(context.OrgId) == 0 {
		log.Fatal("Missing org-id")
	}

	if len(context.ProjectId) == 0 {
		log.Fatal("Missing project-id")
	}

	return context
}

func tryFillFromProjectSettings(context *cb.CloudBuildContext) {
	baseName := path.Join("ProjectSettings", "ProjectSettings.asset")
	filename := tryFindFile(context, baseName)

	if filename == "" {
		return
	}

	d, err := ioutil.ReadFile(filename)
	if err != nil {
		if context.Verbose {
			log.Print(err)
		}
		return
	}

	var settings projectSettings
	err = yaml.Unmarshal(d, &settings)
	if err != nil {
		if context.Verbose {
			log.Print(err)
		}
		return
	}

	if context.Verbose {
		log.Printf("Discovered Unity settings: %s", filename)
	}

	if context.OrgId == "" {
		context.OrgId = settings.PlayerSettings.OrgId
	}
	if context.ProjectId == "" {
		context.ProjectId = settings.PlayerSettings.ProjectId
	}
}

func tryFindFile(context *cb.CloudBuildContext, baseFilename string) string {
	cwd, err := os.Getwd()
	if err != nil {
		if context.Verbose {
			log.Print(err)
		}
		return ""
	}

	filename := path.Join(cwd, baseFilename)
	for {
		if info, err := os.Stat(path.Join(cwd, baseFilename)); err != nil && !os.IsNotExist(err) {
			if context.Verbose {
				log.Print(err)
			}
			return ""
		} else if os.IsNotExist(err) {
			newCwd := path.Dir(cwd)
			if newCwd == cwd {
				return ""
			}
			cwd = newCwd
			filename = path.Join(cwd, baseFilename)
			continue
		} else if info.IsDir() {
			if context.Verbose {
				log.Print("Is dir")
			}
			return ""
		}

		break
	}

	return filename
}

type projectSettings struct {
	PlayerSettings struct {
		OrgId     string `yaml:"organizationId"`
		ProjectId string `yaml:"cloudProjectId"`
	} `yaml:"PlayerSettings"`
}
