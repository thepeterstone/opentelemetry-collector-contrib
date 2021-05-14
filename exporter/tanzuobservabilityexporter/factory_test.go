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
	"context"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configtest"
	"go.uber.org/zap"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	require.NoError(t, configcheck.ValidateConfig(cfg))

	actual, ok := cfg.(*Config)
	require.True(t, ok, "invalid Config: %#v", cfg)
	assert.Equal(t, "http://localhost:30001", actual.Traces.Endpoint)
	assert.Equal(t, "defaultApp", actual.Traces.DefaultApplication)
	assert.Equal(t, "defaultService", actual.Traces.DefaultService)
}

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	// TODO come back to config.Type
	factories.Exporters[config.Type(exporterType)] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	actual, ok := cfg.Exporters[config.NewID("tanzuobservability")]
	require.True(t, ok)
	expected := &Config{
		ExporterSettings: config.NewExporterSettings(config.NewID("tanzuobservability")),
		Traces: TracesConfig{
			Endpoint:           "http://localhost:40001",
			DefaultApplication: "an_application",
			DefaultService:     "a_service",
		},
	}
	assert.Equal(t, expected, actual)
}

func TestCreateExporter(t *testing.T) {
	defaultConfig := createDefaultConfig()
	cfg := defaultConfig.(*Config)
	params := component.ExporterCreateParams{Logger: zap.NewNop()}

	te, err := createTraceExporter(context.Background(), params, cfg)
	assert.Nil(t, err)
	assert.NotNil(t, te, "failed to create trace exporter")
}

func TestCreateTraceExporterNilConfigError(t *testing.T) {
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err := createTraceExporter(context.Background(), params, nil)
	assert.Error(t, err)
}
