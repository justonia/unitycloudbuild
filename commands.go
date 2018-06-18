package unitycloudbuild

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type OutputFormat int

const (
	OutputFormat_None OutputFormat = iota
	OutputFormat_JSON
	OutputFormat_Human
)

type CloudBuildContext struct {
	OrgId        string       `json:"orgid"`
	ProjectId    string       `json:"projectid"`
	ApiKey       string       `json:"apikey"`
	OutputFormat OutputFormat `json:"outputformat"`
}

var validPlatforms = []string{
	"ios",
	"android",
	"webgl",
	"standaloneosxintel",
	"standaloneosxintel64",
	"standaloneosxuniversal",
	"standalonewindows",
	"standalonewindows64",
	"standalonelinux",
	"standalonelinux64",
	"standalonelinuxuniversal",
}

var platformShorthand = map[string]string{
	"osx":   "standaloneosxuniversal",
	"win":   "standalonewindows",
	"win64": "standalonewindows64",
	"linux": "standalonelinuxuniversal",
	"":      "",
}

func init() {
	for _, p := range validPlatforms {
		platformShorthand[p] = p
	}
}

func Builds_List(context *CloudBuildContext, buildTargetId string, filterStatus string, filterPlatform string, limit int64) ([]Build, error) {
	client := &http.Client{}
	req := buildRequest(context, fmt.Sprintf("buildtargets/%s/builds", buildTargetId))

	q := req.URL.Query()
	if len(filterStatus) != 0 {
		q.Add("buildStatus", filterStatus)
	}

	if len(filterPlatform) != 0 {
		if val, ok := platformShorthand[filterPlatform]; ok {
			q.Add("platform", val)
		} else {
			log.Fatalf("No such platform: %s", filterPlatform)
		}
	}

	req.URL.RawQuery = q.Encode()

	var entries []Build
	doRequest(client, req, &entries)

	if limit > 0 {
		entries = entries[0:min(len(entries), int(limit))]
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		for _, build := range entries {
			outputBuild(build)
			fmt.Println()
		}
	case OutputFormat_JSON:
		dumpJson(entries)
	}

	return entries, nil
}

func Builds_Latest(context *CloudBuildContext) ([]BuildTarget, error) {
	client := &http.Client{}
	req := buildRequest(context, "buildtargets")

	q := req.URL.Query()
	q.Add("include_last_success", "true")
	req.URL.RawQuery = q.Encode()

	var entries []BuildTarget
	doRequest(client, req, &entries)

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		for _, target := range entries {
			if len(target.Builds) > 0 {
				build := target.Builds[0]
				outputBuild(build)
			} else {
				fmt.Printf("Target: %s\n", target.Id)
				fmt.Printf("  <No builds successful>")
			}

			fmt.Println()
		}

	case OutputFormat_JSON:
		dumpJson(entries)
	}

	return entries, nil
}

func Targets_List(context *CloudBuildContext) error {
	client := &http.Client{}
	req := buildRequest(context, "buildtargets")

	q := req.URL.Query()
	q.Add("include", "settings")
	req.URL.RawQuery = q.Encode()

	var entries []BuildTarget
	doRequest(client, req, &entries)

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		for _, target := range entries {
			fmt.Printf("Target: %s\n", target.Name)
			fmt.Printf("  ID:        %s\n", target.Id)
			fmt.Printf("  Enabled:   %v\n", target.Enabled)
			fmt.Printf("  AutoBuild: %v\n", target.Settings.AutoBuild)
			fmt.Printf("  Branch:    %s\n", target.Settings.Scm.Branch)
			fmt.Printf("  Unity:     %s\n", strings.Replace(target.Settings.UnityVersion, "_", ".", -1))
			fmt.Println()
		}

	case OutputFormat_JSON:
		dumpJson(entries)
	}

	return nil
}

func outputBuild(build Build) {
	fmt.Printf("Target: %s, (Build #%d)\n", build.TargetId, build.Number)
	fmt.Printf("  Status:   %s\n", build.Status)
	fmt.Printf("  Time:     %v\n", time.Second*time.Duration(build.TotalTimeSeconds))
	if len(build.LastBuiltRevision) > 0 {
		fmt.Printf("  Revision: %s\n", build.LastBuiltRevision)
	}
	if build.Links.DownloadPrimary != nil {
		fmt.Printf("  Download: %s\n", build.Links.DownloadPrimary.Href)
	}
}

func dumpJson(i interface{}) {
	b, _ := json.MarshalIndent(i, "", "    ")
	fmt.Println(string(b))
}

func doRequest(client *http.Client, req *http.Request, result interface{}) {
	resp, err := client.Do(req)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if err = json.Unmarshal(body, &result); err != nil {
		log.Fatal(err)
	}
}

func buildRequest(context *CloudBuildContext, path string) *http.Request {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://build-api.cloud.unity3d.com/api/v1/orgs/%s/projects/%s/%s", context.OrgId, context.ProjectId, path), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth("", context.ApiKey)
	return req
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
