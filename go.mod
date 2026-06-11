module github.com/vpptu/ruleflow

go 1.25.0

require github.com/vpptu/ruleflow/pkg/ruleflow/config v0.0.0

require google.golang.org/grpc v1.81.1

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/vpptu/ruleflow/pkg/ruleflow/adapter => ./pkg/ruleflow/adapter
	github.com/vpptu/ruleflow/pkg/ruleflow/config => ./pkg/ruleflow/config
	github.com/vpptu/ruleflow/pkg/ruleflow/contrib/circuitbreaker => ./pkg/ruleflow/contrib/circuitbreaker
	github.com/vpptu/ruleflow/pkg/ruleflow/contrib/otel => ./pkg/ruleflow/contrib/otel
	github.com/vpptu/ruleflow/pkg/ruleflow/contrib/prometheus => ./pkg/ruleflow/contrib/prometheus
	github.com/vpptu/ruleflow/pkg/ruleflow/ext => ./pkg/ruleflow/ext
)
