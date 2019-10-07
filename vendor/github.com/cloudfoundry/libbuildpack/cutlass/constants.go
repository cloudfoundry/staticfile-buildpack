package cutlass

import (
	"bytes"
	"io"
)

var (
	DefaultMemory       string
	DefaultDisk         string
	Cached              bool
	DefaultStdoutStderr io.Writer = &bytes.Buffer{}
)
