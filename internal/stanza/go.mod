module github.com/open-telemetry/opentelemetry-collector-contrib/internal/stanza

go 1.16

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage v0.0.0-00010101000000-000000000000
	github.com/open-telemetry/opentelemetry-log-collection v0.19.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.29.1-0.20210708180338-a4354b6e8e39
	go.opentelemetry.io/collector/model v0.0.0-00010101000000-000000000000
	go.uber.org/multierr v1.6.0
	go.uber.org/zap v1.18.1
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage => ../../extension/storage

replace go.opentelemetry.io/collector/model => go.opentelemetry.io/collector/model v0.0.0-20210708180338-a4354b6e8e39
