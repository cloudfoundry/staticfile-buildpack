package cutlass

import (
	"os"

	"code.cloudfoundry.org/lager"
)

var DefaultLogger = NewLogger()

func NewLogger() lager.Logger {
	logger := lager.NewLogger("cutlass")
	if os.Getenv("CUTLASS_DEBUG") != "" {
		logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.DEBUG))
	}

	return logger
}
