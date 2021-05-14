// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tanzuobservabilityexporter

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
)

func TestConfigRequiresNonEmptyEndpoint(t *testing.T) {
	c := &Config{
		ExporterSettings: config.ExporterSettings{},
		Traces: TracesConfig{
			Endpoint:           "",
			DefaultApplication: "application",
			DefaultService:     "service",
		},
	}

	require.Error(t, c.Validate())
}

func TestConfigRequiresNonEmptyDefaultApplication(t *testing.T) {
	c := &Config{
		ExporterSettings: config.ExporterSettings{},
		Traces: TracesConfig{
			Endpoint:           "http://localhost:8080",
			DefaultApplication: "",
			DefaultService:     "service",
		},
	}

	require.Error(t, c.Validate())
}

func TestConfigRequiresNonEmptyDefaultService(t *testing.T) {
	c := &Config{
		ExporterSettings: config.ExporterSettings{},
		Traces: TracesConfig{
			Endpoint:           "http://localhost:8080",
			DefaultApplication: "application",
			DefaultService:     "",
		},
	}

	require.Error(t, c.Validate())
}
