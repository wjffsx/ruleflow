module github.com/vpptu/ruleflow/pkg/ruleflow/ext

go 1.25.0

require (
	github.com/expr-lang/expr v1.17.8
	github.com/vpptu/ruleflow v0.0.0
	github.com/vpptu/ruleflow/pkg/ruleflow/adapter v0.0.0
)

replace (
	github.com/vpptu/ruleflow => ../../..
	github.com/vpptu/ruleflow/pkg/ruleflow/adapter => ../adapter
)
