module github.com/legible-money/legible/tools/agent-swarm

go 1.22

require (
	github.com/legible-money/legible/tools/agent-sdk v0.0.0
	github.com/legible-money/legible/tools/picoclaw-plugin v0.0.0
)

replace (
	github.com/legible-money/legible/tools/agent-sdk => ../agent-sdk
	github.com/legible-money/legible/tools/picoclaw-plugin => ../picoclaw-plugin
)
