#!/bin/bash
set -e

# linux
env GOOS=linux GOARCH=amd64 go build github.com/justonia/unitycloudbuild/unity-cb-tool
zip -r unity-cb-tool.linux.zip unity-cb-tool
rm unity-cb-tool

# osx
env GOOS=darwin GOARCH=amd64 go build github.com/justonia/unitycloudbuild/unity-cb-tool
zip -r unity-cb-tool.osx.zip unity-cb-tool
rm unity-cb-tool

# windows
env GOOS=windows GOARCH=amd64 go build github.com/justonia/unitycloudbuild/unity-cb-tool
zip -r unity-cb-tool.windows.zip unity-cb-tool.exe
rm unity-cb-tool.exe

