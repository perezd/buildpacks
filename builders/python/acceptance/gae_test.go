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
package acceptance

import (
	"testing"

	"github.com/GoogleCloudPlatform/buildpacks/internal/acceptance"
)

func init() {
	acceptance.DefineFlags()
}

func TestAcceptance(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.Test{
		{
			App: "no_requirements_txt",
		},
		{
			App:             "requirements_txt",
			MustNotOutput:   []string{`WARNING: You are using pip version`},
			EnableCacheTest: true,
		},
		{
			App: "requirements_bin_conflict",
		},
		{
			App: "requirements_builtin_conflict",
		},
		{
			App:             "pip_dependency",
			EnableCacheTest: true,
		},
		{
			App: "gunicorn_present",
		},
		{
			App: "gunicorn_outdated",
		},
		{
			App: "custom_entrypoint",
			Env: []string{"GOOGLE_ENTRYPOINT=uwsgi --http :$PORT --wsgi-file custom.py --callable app"},
		},
		{
			Name: "custom gunicorn entrypoint",
			App:  "gunicorn_present",
			Env:  []string{"GOOGLE_ENTRYPOINT=gunicorn main:app"},
		},
		// Test that we get a warning when GAE_APP_ENGINE_APIS is set but no lib is used.
		{
			Name:       "GAE_APP_ENGINE_APIS set with no use",
			App:        "no_requirements_txt",
			Env:        []string{"GAE_APP_ENGINE_APIS=TRUE"},
			MustOutput: []string{"App Engine APIs are enabled, but don't appear to be used, causing a possible performance penalty. Delete app_engine_apis from your app.yaml."},
		},
		// Test that we get a warning using SDK libraries without setting flag.
		{
			Name:       "appengine_sdk dependencies without flag",
			App:        "appengine_sdk",
			MustOutput: []string{"There is a dependency on App Engine APIs, but they are not enabled in your app.yaml. Set the app_engine_apis property."},
		},
	}

	for _, tc := range testCases {
		tc := applyStaticTestOptions(tc)
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			acceptance.TestApp(t, builderImage, runImage, tc)
		})
	}
}

func applyStaticTestOptions(tc acceptance.Test) acceptance.Test {
	if tc.Name == "" {
		tc.Name = tc.App
	}
	tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")
	return tc
}

func TestFailures(t *testing.T) {
	builderImage, runImage, cleanup := acceptance.ProvisionImages(t)
	t.Cleanup(cleanup)

	testCases := []acceptance.FailureTest{
		{
			Name: "conflicting dependencies",
			App:  "pip_check",
			// The second warning message is cut short because it's not deterministic.
			MustMatch: `(Cannot install diamond-dependency because these package versions have conflicting dependencies.|found incompatible dependencies: "sub-dependency-)`,
		},
	}

	for _, tc := range testCases {
		tc.Env = append(tc.Env, "X_GOOGLE_TARGET_PLATFORM=gae")
		tc := tc
		t.Run(tc.App, func(t *testing.T) {
			t.Parallel()
			tc.Env = append(tc.Env, "GOOGLE_RUNTIME=python310", "X_GOOGLE_TARGET_PLATFORM=gae")
			acceptance.TestBuildFailure(t, builderImage, runImage, tc)
		})
	}
}
