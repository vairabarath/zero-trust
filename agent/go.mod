module github.com/ztna-system/agent

go 1.21

require (
	github.com/ztna-system/proto v0.0.0
	google.golang.org/grpc v1.65.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240730163845-b1a4ccb954bf // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/ztna-system/proto => ../proto
