# Unity Cloud Build Tools
CLI tool for interacting with Unity Cloud Build. 

This tool is not meant to be an exhaustive wrapper around every single Cloud Build endpoint,
but instead it is meant to provide a quick way to accomplish common tasks.

See [Releases](https://github.com/justonia/unitycloudbuild/releases) for 64-bit Windows, Mac, and Linux binaries.

```
NAME:
   unity-cb-tool - A tool to interact with Unity Cloud Build

USAGE:
   unity-cb-tool [global options] command [command options] [arguments...]

VERSION:
   0.2.3

COMMANDS:
     builds   
     targets  
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --api-key value     Unity API key [$UNITY_API_KEY]
   --org-id value      Unity Organization ID [$UNITY_ORG_ID]
   --project-id value  Unity Project ID [$UNITY_PROJECT_ID]
   --json              If true, output responses in JSON
   --help, -h          show help
   --version, -v       print the version
```

By default, commands output human-readable data. If --json is specified as a root flag
a more detailed JSON response will be outputted (e.g. `unity-cb-tool --json targets list`).

**NOTE:** In the examples below, the two target IDs 'windows-x64' and 'macos' are from 
one of my projects. The IDs for your project will be whatever you have setup for build
targets in Cloud Build. The easiest way to find your target IDs is to run `unity-cb-tool targets list`.

## Configuration

This tool requires you to specify your API key, organization ID, and project ID for any of
the commands to work. 

If the organization ID and project ID are not in the environment variables or not provided
as an argument, the tool will attempt to locate your ProjectSettings.asset file and pull 
those values from it.

Another way to provide these config values are to specify them as environment variables:
* API Key: `UNITY_API_KEY`
* Org ID: `UNITY_ORG_ID`
* Project ID: `UNITY_PROJECT_ID`

Alternatively you could specify each explicitly, e.g.:

```
unity-cb-tool --api-key MYAPIKEY --org-id my-org-id --project-id MYPROJECTID builds latest
```

Or via a combination of environment variables and explicit specification:

```
# Could stick these in your .bashrc or Windows environment
export UNITY_API_KEY=MYAPIKEY
export UNITY_ORG_ID=my-org-id

unity-cb-tool --project-id MYPROJECTID builds latest
```

## Scripting Example

Here's a Bash script I use to kick off all platform builds and then download content into
the directories used by the Steam upload tool.

```
#!/bin/bash

# Error out if any command fails
set -e -x

# Clean out existing files in Steam depot content directories
rm -rf Steam/windows_content/*
touch Steam/windows_content/.placeholder
rm -rf Steam/mac_content/*
touch Steam/mac_content/.placeholder

# Stop any existing builds
./unity-cb-tool builds cancel --all

# Start builds for all enabled targets
./unity-cb-tool builds start --all

# Wait for all builds to complete. If one fails, the script will abort.
./unity-cb-tool builds wait-for-complete --all --abort-on-fail

# Sanity check, make sure the new builds match the local Git commit.
./unity-cb-tool git build-matches-head --all

# Download and unzip the builds into the Steam depot content directories
./unity-cb-tool builds download -t windows-x64 --latest --unzip -o Steam/windows_content
./unity-cb-tool builds download -t macos --latest --unzip -o Steam/mac_content
```

## Commands

### `targets list`

```
NAME:
   unity-cb-tool targets list - List all build targets

USAGE:
   unity-cb-tool targets list [arguments...]
```

#### Example

```
unity-cb-tool targets list

---

Target: Windows x64
  ID:        windows-x64
  Enabled:   true
  AutoBuild: true
  Branch:    release
  Unity:     2018.1.2f1

Target: MacOS
  ID:        macos
  Enabled:   true
  AutoBuild: true
  Branch:    release
  Unity:     2018.1.2f1
```

### `builds list`

```
NAME:
   unity-cb-tool builds list - List builds

USAGE:
   unity-cb-tool builds list [command options] [arguments...]

OPTIONS:
   --target-id value        Specific target ID or _all for all targets (default: "_all")
   --filter-status value    (queued, sentToBuilder, started, restarted, success, failure, canceled, unknown)
   --filter-platform value  (ios, android, webgl, osx, win, win64, linux)
   --limit value, -l value  If >0 show only the specified number of builds (default: 0)
```

#### Example
```
unity-cb-tool builds list --limit 10

---

Target: windows-x64, (Build #16)
  Status:   success
  Time:     12m15s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...

Target: macos, (Build #15)
  Status:   success
  Time:     18m1s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...

(truncated...)
```

### `builds latest`

```
NAME:
   unity-cb-tool builds latest - List latest builds for every build target

USAGE:
   unity-cb-tool builds latest [command options] [arguments...]

OPTIONS:
   --success  If true, only show latest successful build
```

#### Examples

Get all the latest builds of any status.
```
unity-cb-tool builds latest

---

Target: macos, (Build #25)
  Created:  2018-06-19 18:52:56.402 +0000 UTC
  Status:   canceled
  Time:     17s

Target: Windows x64 (id=windows-x64)
  Build:    #16
  Status:   success
  Time:     17m13s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...

```

Get all the latest successful builds.
```
unity-cb-tool builds latest --success

---

Target: Windows x64 (id=windows-x64)
  Build:    #16
  Status:   success
  Time:     17m13s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...

Target: MacOS (id=macos)
  Build:    #15
  Status:   success
  Time:     18m3s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...
```

### `builds status`

```
NAME:
   unity-cb-tool builds status - Retrieve status of a build

USAGE:
   unity-cb-tool builds status [command options] [arguments...]

OPTIONS:
   --target-id value, -t value  Build target ID
   --build value, -b value      Build number for build target (default: -1)
   
```

#### Examples

```
unity-cb-tool builds status -t windows-x64 -b 16

---

Target: Windows x64 (id=windows-x64)
  Build:    #16
  Status:   success
  Time:     17m13s
  Revision: 9102ca18b98706193a6b9d92d51cab8928bd7b97
  Download: https://unitycloud-build-user-svc-live-build.s3.amazonaws.com/...

```

### `builds start`

```
NAME:
   unity-cb-tool builds start - Start a build for a build target, or if --all is specified start builds for all enabled targets

USAGE:
   unity-cb-tool builds start [command options] [arguments...]

OPTIONS:
   --all                        If true, start builds on all enabled targets
   --clean                      Force a clean build.
   --target-id value, -t value  Build target ID
   
```

#### Examples

Start a build for a specific target.
```
unity-cb-tool builds start -t windows-x64

---

Target: windows-x64, (Build #33)
  Created:  2018-06-19 18:52:02.951 +0000 UTC
  Status:   queued
  Time:     0s

```

Start a build for all enabled targets.
```
unity-cb-tool builds start --all

---

Target: windows-x64, (Build #34)
  Created:  2018-06-19 18:52:56.397 +0000 UTC
  Status:   queued
  Time:     0s

Target: macos, (Build #25)
  Created:  2018-06-19 18:52:56.402 +0000 UTC
  Status:   queued
  Time:     0s

```

### `builds cancel`

```
NAME:
   unity-cb-tool builds cancel - Cancel a build for a build target, or if --all is specified cancel all builds

USAGE:
   unity-cb-tool builds cancel [command options] [arguments...]

OPTIONS:
   --all                        If true, cancel all builds
   --target-id value, -t value  Build target ID
   --build value, -b value      Build number for build target (default: -1)
```

#### Examples

Cancel a specific build.
```
unity-cb-tool builds cancel -t windows-x64 -b 17

---

(no output)
```

Cancel all builds for a specific target.
```
unity-cb-tool builds cancel -t windows-x64 --all

---

(no output)
```

Cancel all builds for all targets.
```
unity-cb-tool builds cancel --all

---

(no output)
```

### `builds download`

```
NAME:
   unity-cb-tool builds download - Download builds

USAGE:
   unity-cb-tool builds download [command options] [arguments...]

OPTIONS:
   --target-id value, -t value  Build target ID
   --build value, -b value      Build number for build target (default: -1)
   --latest                     If true, download the latest successful build
   --output value, -o value     If set, the build is written to this directory instead
   
```

#### Examples

Download a specific build.
```
unity-cb-tool builds download -t windows-x64 -b 30 -o Builds/

---

Downloading to: Builds/second-wind-interactive-dntm-windows-x64-30.zip
Download complete.
```

Download the latest build for a target.
```
unity-cb-tool builds download -t macos --latest -o Builds/

---

Latest build is #22.
Downloading to: Builds/second-wind-interactive-dntm-macos-22.zip
Download complete.
```

Download the latest build for a target and unzip its contents.
```
unity-cb-tool builds download -t windows-x64 --latest -o Builds/ --unzip

---

Latest build is #30.
Downloading to: /tmp/second-wind-interactive-dntm-windows-x64-30.zip124929410
Download complete.
Unzipping content to: Builds/
Writing: Builds/DNTM.exe
Writing: Builds/DNTM_Data/app.info
Writing: Builds/DNTM_Data/boot.config
Writing: Builds/DNTM_Data/globalgamemanagers
Writing: Builds/DNTM_Data/globalgamemanagers.assets
Writing: Builds/DNTM_Data/level0
Writing: Builds/DNTM_Data/level1
Writing: Builds/DNTM_Data/level2
Writing: Builds/DNTM_Data/level3

(truncated)
```

### `builds wait-for-complete`

Wait for builds to complete. If any build fails or is canceled the exit code will be 1.

```
NAME:
   unity-cb-tool builds wait-for-complete - Wait for in-progress build(s) to finish

USAGE:
   unity-cb-tool builds wait-for-complete [command options] [arguments...]

OPTIONS:
   --target-id value, -t value  Build target ID
   --build value, -b value      Build number for build target (default: -1)
   --all                        If true, wait for all active builds for all enabled targets
   --abort-on-fail              If true, and --all is specified, exit as soon as one build fails or is canceled.
```

#### Examples

Wait for a single build to finish.
```
unity-cb-tool builds wait-for-complete -t windows-x64 -b 8 

---

Watching: windows-x64 #8
Build: windows-x64 #8 status changed from sentToBuilder to started

(...time elapses)

Build: windows-x64 #8 status changed from started to success
Build: windowx-x64 #8 finished.
Build(s) complete.

```

Wait for the latest build for a target to finish.
```
unity-cb-tool builds wait-for-complete -t windows-x64

---

Watching: windows-x64 #9
Build: windows-x64 #9 status changed from sentToBuilder to started

(...time elapses)

Build: windows-x64 #9 status changed from started to success
Build: windowx-x64 #9 finished.
Build(s) complete.

```

Wait for the latest builds in all enabled targets to finish.
```
unity-cb-tool builds wait-for-complete --all

---

Watching: windows-x64 #9
Watching: macos #4
Build: macos #4 status changed from sentToBuilder to started
Build: windows-x64 #9 status changed from sentToBuilder to started

(...time elapses)

Build: macos #4 status changed from started to success.
Build: macos #4 finished.

(...time elapses)

Build: windows-x64 #9 status changed from started to success
Build: windows-x64 #9 finished.
Build(s) complete.

```

Wait for the latest builds in all enabled targets to finish, and abort if any one of them fails for any reason.
```
unity-cb-tool builds wait-for-complete --all --abort-on-fail

---

Watching: windows-x64 #13
Watching: macos #9
Build: macos #9 status changed from sentToBuilder to started
Build: windows-x64 #13 status changed from sentToBuilder to started

(...time elapses)

Build: macos #9 status changed from started to failure
Build: macos #9 failed with status: failure
Aborting early, build: macos #9 failed with status: canceled
```

### `git head`

Prints info about the current commit, if a Git repo is found in the current directory or any parent directory.

```
NAME:
   unity-cb-tool git head - Output current revision and commit message for HEAD

USAGE:
   unity-cb-tool git head [command options] [arguments...]

OPTIONS:
   --repo-path value, -p value  If set, search for Git repo there instead of current working directory
```

#### Example

```
unity-cb-tool git head

---

Revision: d1396dfaddbaf0b9294d3b7509bd6ae8fc2a18fd
Message:  Committed some stuff.

```

### `git build-matches-head`

Checks if builds match the current HEAD revision. Exit code 1 is returned if any build does not match.

```
NAME:
   unity-cb-tool git build-matches-head - Determine if the build(s) match the current HEAD revision

USAGE:
   unity-cb-tool git build-matches-head [command options] [arguments...]

OPTIONS:
   --target-id value, -t value  Build target ID
   --build value, -b value      Build number for build target (default: -1)
   --all                        If true, check if all enabled targets match
   --repo-path value, -p value  If set, search for Git repo there instead of current working directory
```

#### Examples

Check if a specific build matches.
```
unity-cb-tool git build-matches-head -t windows-x64 -b 1

---

HEAD: 82eee0b975ede2c4b71b780fb77ca2987b830ec0
Build windows-x64 #1 matches HEAD.
```

Check if a specific build matches (it doesn't).
```
unity-cb-tool git build-matches-head -t windows-x64 -b 4

---

HEAD: 82eee0b975ede2c4b71b780fb77ca2987b830ec0
Build windows-x64 #4 is revision 324dfs3f, head is 82eee0b9
Build(s) do not match.
```

Check if latest build for a target matches.
```
unity-cb-tool git build-matches-head -t windows-x64

---

HEAD: 82eee0b975ede2c4b71b780fb77ca2987b830ec0
Build windows-x64 #19 matches HEAD.
```

Check if all latest builds for all enabled targets match.
```
unity-cb-tool git build-matches-head --all

---

HEAD: 82eee0b975ede2c4b71b780fb77ca2987b830ec0
Build windows-x64 #13 matches HEAD.
Build macos #10 matches HEAD.
```

Check if all latest builds for all enabled targets match (one target missing build).
```
unity-cb-tool git build-matches-head --all

---

HEAD: 82eee0b975ede2c4b71b780fb77ca2987b830ec0
Target default-linux-desktop-universal does not have a successful build.
Build windows-x64 #13 matches HEAD.
Build macos #10 matches HEAD.
Build(s) do not match.
```
