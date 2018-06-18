# UnityCloudBuildTool
CLI tool for interacting with Unity Cloud Build

```
NAME:
   unity-cb-tool - A new cli application

USAGE:
   unity-cb-tool [global options] command [command options] [arguments...]

VERSION:
   0.1.0

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

## builds 
```
NAME:
   unity-cb-tool builds - 

USAGE:
   unity-cb-tool builds command [command options] [arguments...]

COMMANDS:
     list    List builds
     latest  List latest builds for every build target

OPTIONS:
   --help, -h  show help
```

### builds list

```
NAME:
   unity-cb-tool builds list - List builds

USAGE:
   unity-cb-tool builds list [command options] [arguments...]

OPTIONS:
   --target-id value        Specific target ID or _all for all targets (default: "_all")
   --filter-status value    (queued, sentToBuilder, started, restarted, success, failure, canceled, unknown)
   --filter-platform value  (ios, android, webgl, osx, win, win64)
   --limit value, -l value  If >0 show only the specified number of builds (default: 0)
```

### builds latest

```
NAME:
   unity-cb-tool builds latest - List latest builds for every build target

USAGE:
   unity-cb-tool builds latest [arguments...]
```

## targets

```
NAME:
   unity-cb-tool targets - 

USAGE:
   unity-cb-tool targets command [command options] [arguments...]

COMMANDS:
     list  List all build targets

OPTIONS:
   --help, -h  show help
```

### targets list

```
NAME:
   unity-cb-tool targets list - List all build targets

USAGE:
   unity-cb-tool targets list [arguments...]
```
