# Unity Cloud Build Tools
CLI tool for interacting with Unity Cloud Build.

This tool is not meant to be an exhaustive wrapper around every single Cloud Build endpoint,
but instead it is meant to provide a quick way to accomplish common tasks.

```
NAME:
   unity-cb-tool - A new cli application

USAGE:
   unity-cb-tool [global options] command [command options] [arguments...]

VERSION:
   0.2.1

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

The easiest way to provide these data are to specify them as environment variables:
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

I will add support for a config file at some indeterminate point in the future. See [Issue #2](https://github.com/justonia/unitycloudbuild/issues/2).

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

