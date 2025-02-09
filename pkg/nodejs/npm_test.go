// Copyright 2021 Google LLC
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

package nodejs

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/pkg/gcpbuildpack"
)

func TestRequestedNPMVersion(t *testing.T) {
	testCases := []struct {
		name        string
		packageJSON string
		want        string
		wantErr     bool
	}{
		{
			name:        "default is empty",
			packageJSON: `{}`,
			want:        "",
		},
		{
			name:        "engines.npm set",
			packageJSON: `{"engines": {"npm": "2.2.2"}}`,
			want:        "2.2.2",
		},
		{
			name:        "invalid package.json",
			packageJSON: `invalid json`,
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			dir := t.TempDir()
			if tc.packageJSON != "" {
				path := filepath.Join(dir, "package.json")
				if err := ioutil.WriteFile(path, []byte(tc.packageJSON), 0744); err != nil {
					t.Fatalf("writing %s: %v", path, err)
				}
			}

			got, err := RequestedNPMVersion(dir)
			if tc.wantErr == (err == nil) {
				t.Errorf("RequestedNPMVersion(%q) got error: %v, want err? %t", dir, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("RequestedNPMVersion(%q) = %q, want %q", dir, got, tc.want)
			}
		})
	}
}

func TestNPMInstallCommand(t *testing.T) {
	testCases := []struct {
		version string
		want    string
	}{
		{
			version: "8.3.1",
			want:    "ci",
		},
		{
			version: "5.7.0",
			want:    "install",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) string) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) string { return tc.version }

			got, err := NPMInstallCommand(nil)
			if err != nil {
				t.Fatalf("Node.js %v: NPMInstallCommand(nil) got error: %v", tc.version, err)
			}
			if got != tc.want {
				t.Errorf("Node.js %v: NPMInstallCommand(nil) = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}

func TestSupportsNPMPrune(t *testing.T) {
	testCases := []struct {
		version string
		want    bool
	}{
		{
			version: "8.3.1",
			want:    true,
		},
		{
			version: "5.7.0",
			want:    true,
		},
		{
			version: "5.0.1",
			want:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			defer func(fn func(*gcpbuildpack.Context) string) { npmVersion = fn }(npmVersion)
			npmVersion = func(*gcpbuildpack.Context) string { return tc.version }

			got, err := SupportsNPMPrune(nil)
			if err != nil {
				t.Errorf("npm %v: SupportsNPMPrune(nil) got error: %v", tc.version, err)
			}
			if got != tc.want {
				t.Errorf("npm %v: SupportsNPMPrune(nil) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}
