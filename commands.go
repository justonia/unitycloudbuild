package unitycloudbuild

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
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
	Verbose      bool
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

func Builds_Start(context *CloudBuildContext, buildTargetId string, clean bool) (*BuildAttempt, error) {
	client := &http.Client{}
	req := buildRequest(
		context, "POST", fmt.Sprintf("buildtargets/%s/builds", buildTargetId),
		struct {
			clean bool
		}{clean: clean})

	var entries []BuildAttempt
	doRequest(context, client, req, &entries)

	if entries == nil || len(entries) == 0 {
		return nil, fmt.Errorf("No builds started...")
	} else if len(entries[0].Error) > 0 {
		return nil, fmt.Errorf(entries[0].Error)
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		outputBuild(entries[0].Build)
		fmt.Println()
	case OutputFormat_JSON:
		dumpJson(entries[0])
	}

	return &entries[0], nil
}

func Builds_StartAll(context *CloudBuildContext, clean bool) ([]BuildAttempt, error) {
	client := &http.Client{}
	req := buildRequest(
		context, "POST", "buildtargets/_all/builds",
		struct {
			clean bool
		}{clean: clean})

	var entries []BuildAttempt
	doRequest(context, client, req, &entries)

	if entries == nil || len(entries) == 0 {
		return nil, fmt.Errorf("No builds started...")
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		for _, buildAttempt := range entries {
			if len(buildAttempt.Error) > 0 {
				fmt.Printf("Target: %s\n", buildAttempt.Build.TargetId)
				fmt.Printf("  Error: %s\n", buildAttempt.Error)
			} else {
				outputBuild(buildAttempt.Build)
			}
			fmt.Println()
		}
	case OutputFormat_JSON:
		dumpJson(entries[0])
	}

	return entries, nil
}

func Builds_Cancel(context *CloudBuildContext, buildTargetId string, buildNumber int64) error {
	client := &http.Client{}
	req := buildRequest(context, "DELETE", fmt.Sprintf("buildtargets/%s/builds/%d", buildTargetId, buildNumber), nil)

	resp := doRequest(context, client, req, nil)
	if resp.StatusCode == 404 {
		log.Fatalf("Cannot find %s build #%d", buildTargetId, buildNumber)
	}

	return nil
}

func Builds_CancelAll(context *CloudBuildContext, buildTargetId string) error {
	/* Right now this correct way of canceling all builds is bugged on the Cloud Build
	   side and returning an HTTP 500 error.

	   So we need to do things the hard way and get all targets, and then for each target
	   call the cancel builds endpoint.

	client := &http.Client{}
	req := buildRequest(context, "DELETE", "buildtargets/_all/builds")

	resp := doRequest(context, client, req, nil)
	if resp.StatusCode == 404 {
		log.Fatalf("Cannot find resource")
	}
	*/

	targetsContext := *context
	targetsContext.OutputFormat = OutputFormat_None
	targets, err := Targets_List(&targetsContext)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	for _, target := range targets {
		// If a specific target has been specified, ignore other targets.
		if len(buildTargetId) > 0 && buildTargetId != target.Id {
			continue
		}

		req := buildRequest(context, "DELETE", fmt.Sprintf("buildtargets/%s/builds", target.Id), nil)
		doRequest(context, client, req, nil)
	}

	return nil
}

func Builds_List(context *CloudBuildContext, buildTargetId string, filterStatus string, filterPlatform string, limit int64) ([]Build, error) {
	client := &http.Client{}
	req := buildRequest(context, "GET", fmt.Sprintf("buildtargets/%s/builds", buildTargetId), nil)

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
	doRequest(context, client, req, &entries)

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

func Builds_Latest(context *CloudBuildContext, onlySuccess bool) (map[string]*Build, error) {
	client := &http.Client{}
	builds := make(map[string]*Build)

	// Get all targets along with builds
	req := buildRequest(context, "GET", "buildtargets", nil)

	q := req.URL.Query()
	q.Add("include_last_success", "true")
	req.URL.RawQuery = q.Encode()

	var targets []BuildTarget
	doRequest(context, client, req, &targets)

	for _, target := range targets {
		if len(target.Builds) > 0 {
			builds[target.Id] = &target.Builds[0]
		} else {
			builds[target.Id] = nil
		}
	}

	if !onlySuccess {
		quietContext := *context
		quietContext.OutputFormat = OutputFormat_None

		for _, target := range targets {
			targetBuilds, err := Builds_List(&quietContext, target.Id, "", "", 1)
			if err != nil {
				log.Fatal(err)
			}
			if len(targetBuilds) > 0 {
				builds[target.Id] = &targetBuilds[0]
			}
		}
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		ids := make([]string, 0, len(builds))
		for id := range builds {
			ids = append(ids, id)
		}

		sort.Strings(ids)

		for _, targetId := range ids {
			build := builds[targetId]
			if build != nil {
				outputBuild(*build)
			} else {
				fmt.Printf("Target: %s\n", targetId)
				fmt.Printf("  <No builds successful>\n")
			}

			fmt.Println()
		}

	case OutputFormat_JSON:
		dumpJson(builds)
	}

	return builds, nil
}

func Targets_List(context *CloudBuildContext) ([]BuildTarget, error) {
	client := &http.Client{}
	req := buildRequest(context, "GET", "buildtargets", nil)

	q := req.URL.Query()
	q.Add("include", "settings")
	req.URL.RawQuery = q.Encode()

	var entries []BuildTarget
	doRequest(context, client, req, &entries)

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

	return entries, nil
}

func outputBuild(build Build) {
	fmt.Printf("Target: %s, (Build #%d)\n", build.TargetId, build.Number)
	fmt.Printf("  Created:  %v\n", build.Created)
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

func doRequest(context *CloudBuildContext, client *http.Client, req *http.Request, result interface{}) *http.Response {
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if context.Verbose {
		log.Print("X-RateLimit-Remaining:", resp.Header.Get("X-RateLimit-Remaining"))
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode == 200 || resp.StatusCode == 202 {
		if result != nil {
			if err = json.Unmarshal(body, &result); err != nil {
				log.Fatal(err)
			}
		}
	} else if resp.StatusCode == 204 || resp.StatusCode == 404 {
		// do nothing
	} else {
		var e errorMessage
		json.Unmarshal(body, &e)
		log.Fatalf("[HTTP %d] %s", resp.StatusCode, e.Error)
	}

	return resp
}

func buildRequest(context *CloudBuildContext, method string, path string, body interface{}) *http.Request {
	var postData io.Reader
	if body != nil {
		d, err := json.Marshal(body)
		if err != nil {
			log.Fatal(err)
		}
		postData = bytes.NewBuffer(d)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("https://build-api.cloud.unity3d.com/api/v1/orgs/%s/projects/%s/%s", context.OrgId, context.ProjectId, path), postData)
	if err != nil {
		log.Fatal(err)
	}

	req.SetBasicAuth("", context.ApiKey)

	if postData != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

type errorMessage struct {
	Error string `json:"error"`
}
