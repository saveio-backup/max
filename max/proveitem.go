package max

type PDPItem interface {
	doPdpCalculation() error
	onFailedPdpCalculation(err error) error
	doPdpSubmission([]byte) (txHash []byte, err error)
	onSuccessfulPdpSubmission() error
	onFailedPdpSubmission(err error) error
	getItemKey() string // fileHash for file, sector id string for sector
	getPdpCalculationHeight() uint32
	getPdpSubmissionHeight() uint32
	shouldSavePdpResult() bool
	getPdpCalculationResult() []byte
}
