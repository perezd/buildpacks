// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Binary dotnet/runtime buildpack detects .NET applications
// and install the corresponding version of .NET runtime.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/dotnet"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/env"
	gcp "github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
	"github.com/GoogleCloudPlatform/buildpacks/pkg/runtime"
	"github.com/buildpacks/libcnb"
)

const (
	sdkLayerName = "sdk"
	versionKey   = "version"
	googleMin22  = "google.min.22"
)

func main() {
	gcp.Main(detectFn, buildFn)
}

func detectFn(ctx *gcp.Context) (gcp.DetectResult, error) {
	if result := runtime.CheckOverride("dotnet"); result != nil {
		return result, nil
	}

	if files := dotnet.ProjectFiles(ctx, "."); len(files) != 0 {
		return gcp.OptIn("found project files: " + strings.Join(files, ", ")), nil
	}

	return gcp.OptOut("no project files or .dll files found"), nil
}

func buildFn(ctx *gcp.Context) error {
	sdkVersion, err := dotnet.GetSDKVersion(ctx)
	if err != nil {
		return err
	}
	isDevMode, err := env.IsDevMode()
	if err != nil {
		return fmt.Errorf("checking if dev mode is enabled: %w", err)
	}
	if err := buildSDKLayer(ctx, sdkVersion, isDevMode); err != nil {
		return fmt.Errorf("building the sdk layer: %w", err)
	}
	return nil
}

func buildSDKLayer(ctx *gcp.Context, version string, isDevMode bool) error {
	// Keep the SDK layer for launch in devmode because we use `dotnet watch`.
	sdkl, err := ctx.Layer(sdkLayerName, gcp.BuildLayer, gcp.CacheLayer, gcp.LaunchLayerIfDevMode)
	if err != nil {
		return fmt.Errorf("creating %v layer: %w", sdkLayerName, err)
	}
	sdkMetaVersion := ctx.GetMetadata(sdkl, versionKey)
	cacheHitValue := fmt.Sprintf("version:%s,devMode:%t", version, isDevMode)
	if cacheHitValue == sdkMetaVersion {
		ctx.CacheHit(sdkLayerName)
		ctx.Logf(".NET SDK cache hit, skipping installation.")
		return nil
	}
	ctx.CacheMiss(sdkLayerName)
	if err := ctx.ClearLayer(sdkl); err != nil {
		return fmt.Errorf("clearing layer %q: %w", sdkl.Name, err)
	}
	if err := dlAndInstallSDK(ctx, sdkl, version, isDevMode); err != nil {
		return err
	}
	ctx.SetMetadata(sdkl, versionKey, cacheHitValue)
	return nil
}

func dlAndInstallSDK(ctx *gcp.Context, sdkl *libcnb.Layer, version string, isDevMode bool) error {
	if _, err := runtime.InstallTarballIfNotCached(ctx, runtime.DotnetSDK, version, sdkl); err != nil {
		return err
	}
	setSDKEnvVars(ctx, sdkl, isDevMode)
	return nil
}

func setSDKEnvVars(ctx *gcp.Context, sdkl *libcnb.Layer, isDevMode bool) {
	if ctx.StackID() == googleMin22 {
		sdkl.BuildEnvironment.Default("DOTNET_SYSTEM_GLOBALIZATION_INVARIANT", "true")
	}
	if isDevMode {
		setSDKEnvVarsDevMode(sdkl)
	} else {
		setSDKEnvVarsForBuild(sdkl)
	}
}

// setSDKEnvVarsDevMode sets the env vars for dev mode. In dev mode, the full
// SDK is present at launch time and the runtime layer is not created.
func setSDKEnvVarsDevMode(sdkl *libcnb.Layer) {
	sdkl.SharedEnvironment.Default("DOTNET_ROOT", sdkl.Path)
	sdkl.SharedEnvironment.Prepend("PATH", string(os.PathListSeparator), sdkl.Path)
	sdkl.LaunchEnvironment.Default("DOTNET_RUNNING_IN_CONTAINER", "true")
}

// setSDKEnvVarsForBuild sets the SDK variables needed at build time. The SDK
// layer is only present for the build and the runtime layer is present in the launch
// image.
func setSDKEnvVarsForBuild(sdkl *libcnb.Layer) {
	sdkl.BuildEnvironment.Default("DOTNET_ROOT", sdkl.Path)
	sdkl.BuildEnvironment.Prepend("PATH", string(os.PathListSeparator), sdkl.Path)
}
