module github.com/wjffsx/ruleflow

go 1.25.0

replace (
	github.com/wjffsx/ruleflow/pkg/ruleflow/adapter => ./pkg/ruleflow/adapter
	github.com/wjffsx/ruleflow/pkg/ruleflow/config => ./pkg/ruleflow/config
	github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/circuitbreaker => ./pkg/ruleflow/contrib/circuitbreaker
	github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/otel => ./pkg/ruleflow/contrib/otel
	github.com/wjffsx/ruleflow/pkg/ruleflow/contrib/prometheus => ./pkg/ruleflow/contrib/prometheus
	github.com/wjffsx/ruleflow/pkg/ruleflow/ext => ./pkg/ruleflow/ext
)

require (
	github.com/wjffsx/ruleflow/pkg/ruleflow/config v0.1.0
	google.golang.org/grpc v1.81.1
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
