module github.com/saveio/max

go 1.14

replace (
	github.com/saveio/carrier => ../carrier
	github.com/saveio/themis => ../themis
	github.com/saveio/themis-go-sdk => ../themis-go-sdk
)

require (
	github.com/ethereum/go-ethereum v1.10.2
	github.com/fatih/color v1.10.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gogo/protobuf v1.3.2
	github.com/jbenet/go-cienv v0.1.0
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/rs/cors v1.7.0
	github.com/saveio/themis v0.0.0-00010101000000-000000000000
	github.com/saveio/themis-go-sdk v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210305035536-64b5b1c73954
	golang.org/x/crypto v0.0.0-20210415154028-4f45737414dc
	golang.org/x/net v0.0.0-20210420210106-798c2154c571
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/cheggaaa/pb.v1 v1.0.28
)
