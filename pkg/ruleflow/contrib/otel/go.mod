module github.com/vpptu/ruleflow/pkg/ruleflow/contrib/otel

go 1.25.0

require (
	github.com/vpptu/ruleflow v0.0.0
	go.opentelemetry.io/otel/trace v1.43.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
)

replace github.com/vpptu/ruleflow => ../../../..
