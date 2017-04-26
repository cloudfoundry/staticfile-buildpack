package ansicleaner

import (
	"io"
	"strings"
)

type bufferCleaner struct {
	w        io.Writer
	replacer *strings.Replacer
}

func New(w io.Writer) io.Writer {
	replacer := strings.NewReplacer("\033[31;1m", "", "\033[33;1m", "", "\033[34;1m", "", "\033[0m", "")
	return &bufferCleaner{w: w, replacer: replacer}
}

func (b *bufferCleaner) Write(bytes []byte) (int, error) {
	return b.replacer.WriteString(b.w, string(bytes))
}
