module github.com/aerospike/aerospike-client-go/v7

go 1.20

require (
	github.com/onsi/ginkgo/v2 v2.16.0
	github.com/onsi/gomega v1.32.0
	github.com/yuin/gopher-lua v1.1.1
	golang.org/x/sync v0.7.0
	google.golang.org/grpc v1.63.3
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20240711041743-f6c9dda6c6da // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/wadey/gocovmerge v0.0.0-20160331181800-b5bfa59ec0ad // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240711142825-46eb208f015d // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract (
	v7.3.0 // `Client.BatchGetOperate` issue
	v7.7.0 // nil deref in tend logic
)
