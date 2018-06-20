package unitycloudbuild

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/src-d/go-git.v4"
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

var RateLimitedError = fmt.Errorf("API rate limit reached")
var ResourceNotFoundError = fmt.Errorf("Resource not found")

func init() {
	for _, p := range validPlatforms {
		platformShorthand[p] = p
	}
}

func fatalIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Git_BuildsMatchHead(context *CloudBuildContext, repoPath string, buildTargetId string, buildNumber int64, all bool) (bool, error) {
	quietContext := *context
	if !context.Verbose {
		quietContext.OutputFormat = OutputFormat_None
	}

	commit, err := Git_Head(&quietContext, repoPath)
	if err != nil {
		return false, err
	}

	if context.OutputFormat == OutputFormat_Human {
		fmt.Printf("HEAD: %s\n", commit.Revision)
	}

	var builds []*Build
	var missingBuilds []string

	if all {
		latestBuilds, err := Builds_Latest(&quietContext, true, true)
		if err != nil {
			return false, err
		}
		for targetName, build := range latestBuilds {
			if build == nil {
				missingBuilds = append(missingBuilds, targetName)
				continue
			}
			builds = append(builds, build)
		}
	} else if buildNumber > 0 {
		build, err := Builds_Status(&quietContext, buildTargetId, buildNumber)
		if err != nil {
			return false, err
		}
		builds = append(builds, build)
	} else {
		latestBuilds, err := Builds_Latest(&quietContext, true, true)
		if err != nil {
			return false, err
		} else if build, ok := latestBuilds[buildTargetId]; ok {
			if build == nil {
				missingBuilds = append(missingBuilds, buildTargetId)
			} else {
				builds = append(builds, build)
			}
		}
	}

	allMatch := true

	for _, targetName := range missingBuilds {
		if context.OutputFormat == OutputFormat_Human {
			fmt.Printf("Target %s does not have a successful build.\n", targetName)
		}

		allMatch = false
	}

	for _, build := range builds {
		if build.Status != "success" {
			if context.OutputFormat == OutputFormat_Human {
				fmt.Printf("Build %s #%d is not a successful build\n", build.TargetId, build.Number)
			}
			allMatch = false
			continue
		}

		if build.LastBuiltRevision != commit.Revision {
			if context.OutputFormat == OutputFormat_Human {
				fmt.Printf("Build %s #%d is revision %s, head is %s\n", build.TargetId, build.Number, build.LastBuiltRevision[:8], commit.Revision[:8])
			}

			allMatch = false
			continue
		}

		if context.OutputFormat == OutputFormat_Human {
			fmt.Printf("Build %s #%d matches HEAD.\n", build.TargetId, build.Number)
		}
	}

	return allMatch, nil
}

func Git_Head(context *CloudBuildContext, repoPath string) (*GitCommit, error) {
	if repoPath == "" {
		repoPath = "."
	}

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})

	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	fatalIfError(err)

	commit, err := repo.CommitObject(head.Hash())
	fatalIfError(err)

	info := &GitCommit{
		Revision: commit.Hash.String(),
		Message:  commit.Message,
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		fmt.Printf("Revision: %s\n", info.Revision)
		fmt.Printf("Message:  %s\n", info.Message)
	case OutputFormat_JSON:
		dumpJson(info)
	}

	return info, nil
}

func Builds_WaitForComplete(context *CloudBuildContext, buildTargetId string, buildNumber int64, all bool, abortOnFail bool) error {
	quietContext := *context
	if !context.Verbose {
		quietContext.OutputFormat = OutputFormat_None
	}

	var builds []*Build

	if all {
		latestBuilds, err := Builds_Latest(&quietContext, false, true)
		if err != nil {
			return err
		}
		for _, build := range latestBuilds {
			if build != nil && IsBuildActive(build) {
				builds = append(builds, build)
			}
		}
	} else if buildNumber > 0 {
		build, err := Builds_Status(&quietContext, buildTargetId, buildNumber)
		if err != nil {
			return err
		}
		builds = append(builds, build)
	} else {
		latestBuilds, err := Builds_Latest(&quietContext, false, true)
		if err != nil {
			return err
		} else if build, ok := latestBuilds[buildTargetId]; ok && build != nil {
			builds = append(builds, build)
		}
	}

	if len(builds) == 0 {
		return fmt.Errorf("No builds found")
	}

	finishedBuilds := make(map[string]bool)
	lastBuildStatus := make(map[string]string)
	for _, build := range builds {
		if !IsBuildActive(build) {
			return fmt.Errorf("Build #%d for target %s is not active", build.Number, build.TargetId)
		}

		finishedBuilds[build.UniqueId()] = false
		lastBuildStatus[build.UniqueId()] = build.Status

		if context.OutputFormat == OutputFormat_Human {
			fmt.Printf("Watching: %s #%d\n", build.TargetId, build.Number)
		}
	}

	pollRate := time.Second * 5
	finishedBuildsCount := 0

Poll:
	for {
		select {
		case <-time.After(pollRate):
			if context.Verbose {
				log.Print("Polling...")
			}

			for i, build := range builds {
				if finishedBuilds[build.UniqueId()] {
					continue
				}

				updatedBuild, err := Builds_Status(&quietContext, build.TargetId, int64(build.Number))
				if err == RateLimitedError {
					log.Print("Rate limit hit, backing off")
					break
				} else if err != nil {
					return err
				}

				builds[i] = updatedBuild
				build = updatedBuild

				if lastStatus := lastBuildStatus[build.UniqueId()]; lastStatus != build.Status {
					if context.OutputFormat == OutputFormat_Human {
						fmt.Printf("Build: %s #%d status changed from %s to %s\n", build.TargetId, build.Number, lastStatus, build.Status)
					}

					lastBuildStatus[build.UniqueId()] = build.Status
				}

				if !IsBuildActive(build) {
					finishedBuilds[build.UniqueId()] = true
					finishedBuildsCount++

					if context.OutputFormat == OutputFormat_Human {
						if build.Status != "success" {
							fmt.Printf("Build: %s #%d failed with status: %s\n", build.TargetId, build.Number, build.Status)
						} else {
							fmt.Printf("Build: %s #%d finished.\n", build.TargetId, build.Number)
						}
					}

					if abortOnFail && build.Status != "success" {
						return fmt.Errorf("Aborting early, build: %s #%d failed with status: %s", build.TargetId, build.Number, build.Status)
					}
				}
			}

			if finishedBuildsCount == len(builds) {
				break Poll
			}
		}
	}

	for _, build := range builds {
		if build.Status != "success" {
			return fmt.Errorf("Build: %s #%d failed", build.TargetId, build.Number)
		}
	}

	if context.OutputFormat == OutputFormat_Human {
		fmt.Printf("Build(s) complete.\n")
	}

	return nil
}

func IsBuildActive(build *Build) bool {
	switch build.Status {
	case "success", "failure", "canceled", "unknown":
		return false
	default:
		return true
	}
}

func Builds_Download(context *CloudBuildContext, buildTargetId string, buildNumber int64, latest bool, outputDir string, unzip bool) error {
	quietContext := *context
	quietContext.OutputFormat = OutputFormat_None

	// Find the build information
	var build *Build
	var err error

	if !latest {
		build, err = Builds_Status(&quietContext, buildTargetId, buildNumber)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		targetBuilds, err := Builds_List(&quietContext, buildTargetId, "success", "", 1)
		if err != nil {
			log.Fatal(err)
		} else if len(targetBuilds) == 0 {
			return fmt.Errorf("No successful build for target %s", buildTargetId)
		}

		build = &targetBuilds[0]

		if context.OutputFormat == OutputFormat_Human {
			fmt.Printf("Latest build is #%d.\n", build.Number)
		}
	}

	if context.Verbose {
		log.Printf("Found build #%d for target %s, status: %s", build.Number, build.TargetId, build.Status)
	}

	if build.Status != "success" {
		return fmt.Errorf("Cannot download build, status is '%s'", build.Status)
	} else if build.Links.DownloadPrimary == nil {
		return fmt.Errorf("Missing download link for build")
	} else if unzip && strings.ToLower(build.Links.DownloadPrimary.Meta.Type) != "zip" {
		return fmt.Errorf("Cannot unzip build, filetype is %s", strings.ToLower(build.Links.DownloadPrimary.Meta.Type))
	}

	// Check output dir status
	if len(outputDir) == 0 {
		outputDir = "."
	}

	outputDirInfo, err := os.Stat(outputDir)
	if os.IsNotExist(err) || !outputDirInfo.IsDir() {
		return fmt.Errorf("Error: %s is not a directory or does not exist", outputDir)
	} else if err != nil {
		return fmt.Errorf("Error stat'ing directory: %v", err)
	}

	// Determine output location
	_url, err := url.Parse(build.Links.DownloadPrimary.Href)
	if err != nil {
		return err
	}

	filename := path.Base(_url.Path)
	if _, params, err := mime.ParseMediaType(_url.Query().Get("response-content-disposition")); err == nil {
		filename = params["filename"]
	}

	var file *os.File
	if !unzip {
		if context.Verbose {
			log.Printf("Using filename: %s\n", filename)
		}

		filename = filepath.Join(outputDir, filename)
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
	} else {
		file, err = ioutil.TempFile("", filename)
		if err != nil {
			return err
		}
	}

	defer file.Close()
	if context.OutputFormat == OutputFormat_Human {
		fmt.Printf("Downloading to: %s\n", file.Name())
	}

	// Download build
	err = grabHttpFile(context, _url, file)
	if err != nil {
		// Deferring to have it happen after file.Close()
		defer func() {
			os.Remove(file.Name())
		}()
		return err
	}

	if context.OutputFormat == OutputFormat_Human {
		fmt.Printf("Download complete.\n")
	}

	if !unzip {
		return nil
	}

	// Handle unzipping, need to abstract out at some point...
	defer func() {
		os.Remove(file.Name())
	}()

	if err := file.Sync(); err != nil {
		return err
	}

	zipReader, err := zip.OpenReader(file.Name())
	if err != nil {
		return err
	}

	if context.OutputFormat == OutputFormat_Human {
		fmt.Printf("Unzipping content to: %s\n", outputDir)
	}

	for _, zippedFile := range zipReader.File {
		filePath := filepath.Join(outputDir, filepath.FromSlash(zippedFile.Name))

		if zippedFile.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, zippedFile.Mode()); err != nil {
				return err
			}
			continue
		} else {
			dir, _ := filepath.Split(filePath)
			if err := os.MkdirAll(dir, outputDirInfo.Mode()); err != nil {
				return err
			}
		}

		if context.OutputFormat == OutputFormat_Human {
			fmt.Println("Writing:", filePath)
		}

		fileReader, err := zippedFile.Open()
		if err != nil {
			return err
		}

		if err = func() error {
			defer fileReader.Close()

			targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zippedFile.Mode())
			if err != nil {
				return err
			}

			defer targetFile.Close()

			if _, err := io.Copy(targetFile, fileReader); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			return err
		}
	}

	return nil
}

func grabHttpFile(context *CloudBuildContext, _url *url.URL, dst io.Writer) error {
	if context.Verbose {
		log.Println(_url)
	}

	resp, err := http.Get(_url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if context.Verbose {
		for name, val := range resp.Header {
			log.Printf("Response header: %s=%s", name, val)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Could not download, got status code: %d", resp.StatusCode)
	}

	if _, err = io.Copy(dst, resp.Body); err != nil {
		return err
	}

	return nil
}

func Builds_Start(context *CloudBuildContext, buildTargetId string, clean bool) (*BuildAttempt, error) {
	client := &http.Client{}
	req := buildRequest(
		context, "POST", fmt.Sprintf("buildtargets/%s/builds", buildTargetId),
		struct {
			clean bool
		}{clean: clean})

	var entries []BuildAttempt
	_, err := doRequest(context, client, req, &entries)

	if err != nil {
		return nil, err
	} else if entries == nil || len(entries) == 0 {
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

	_, err := doRequest(context, client, req, &entries)
	if err != nil {
		return nil, err
	}

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

	_, err := doRequest(context, client, req, nil)
	if err == ResourceNotFoundError {
		return fmt.Errorf("Cannot find %s build #%d", buildTargetId, buildNumber)
	} else if err != nil {
		return err
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
		_, err := doRequest(context, client, req, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func Builds_Status(context *CloudBuildContext, buildTargetId string, buildNumber int64) (*Build, error) {
	client := &http.Client{}
	req := buildRequest(context, "GET", fmt.Sprintf("buildtargets/%s/builds/%d", buildTargetId, buildNumber), nil)

	var build Build
	_, err := doRequest(context, client, req, &build)
	if err != nil {
		return nil, err
	}

	switch context.OutputFormat {
	case OutputFormat_None:
		// do nothing
	case OutputFormat_Human:
		outputBuild(build)
		fmt.Println()
	case OutputFormat_JSON:
		dumpJson(build)
	}

	return &build, nil
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

	_, err := doRequest(context, client, req, &entries)
	if err != nil {
		return nil, err
	}

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

func Builds_Latest(context *CloudBuildContext, onlySuccess bool, onlyEnabled bool) (map[string]*Build, error) {
	client := &http.Client{}
	builds := make(map[string]*Build)

	// Get all targets along with builds
	req := buildRequest(context, "GET", "buildtargets", nil)

	q := req.URL.Query()
	q.Add("include_last_success", "true")
	req.URL.RawQuery = q.Encode()

	var targets []BuildTarget

	_, err := doRequest(context, client, req, &targets)
	if err != nil {
		return nil, err
	}

	for _, target := range targets {
		if onlyEnabled && !target.Enabled {
			continue
		}

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
			if onlyEnabled && !target.Enabled {
				continue
			}

			targetBuilds, err := Builds_List(&quietContext, target.Id, "", "", 1)
			if err != nil {
				return nil, err
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

	_, err := doRequest(context, client, req, &entries)
	if err != nil {
		return nil, err
	}

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
	fmt.Printf("  GUID:     %s\n", build.GUID)
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

func doRequest(context *CloudBuildContext, client *http.Client, req *http.Request, result interface{}) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if context.Verbose {
		log.Print("X-RateLimit-Remaining:", resp.Header.Get("X-RateLimit-Remaining"))
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case 429:
		return resp, RateLimitedError
	case 200, 202:
		if result != nil {
			if err = json.Unmarshal(body, &result); err != nil {
				log.Fatal(err)
			}
		}
	case 204:
		// do nothing
	case 404:
		return resp, ResourceNotFoundError
	default:
		var e errorMessage
		json.Unmarshal(body, &e)
		return nil, fmt.Errorf("[HTTP %d] %s", resp.StatusCode, e.Error)
	}

	/*
		if resp.StatusCode == 429 {
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
	*/

	return resp, nil
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
