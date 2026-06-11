module github.com/vpptu/ruleflow/pkg/ruleflow/config

go 1.25.0

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/vpptu/ruleflow v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require golang.org/x/sys v0.42.0 // indirect

replace github.com/vpptu/ruleflow => ../../..
