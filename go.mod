module github.com/saveio/max

go 1.16

require (
	github.com/ethereum/go-ethereum v1.10.15
	github.com/fatih/color v1.10.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gogo/protobuf v1.3.2
	github.com/jbenet/go-cienv v0.1.0
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/rs/cors v1.7.0
	github.com/saveio/dsp-go-sdk v0.0.0-20220928090042-77674439f201
	github.com/saveio/themis v1.0.175
	github.com/saveio/themis-go-sdk v0.0.0-20230314033227-3033a22d3bcd
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	golang.org/x/crypto v0.0.0-20210415154028-4f45737414dc
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/cheggaaa/pb.v1 v1.0.28
)

replace (
	github.com/saveio/carrier => ../carrier
	github.com/saveio/dsp-go-sdk => ../dsp-go-sdk
	github.com/saveio/pylons => ../pylons
	github.com/saveio/scan => ../scan
	github.com/saveio/themis-go-sdk => ../themis-go-sdk
	github.com/saveio/themis => ../themis
)
