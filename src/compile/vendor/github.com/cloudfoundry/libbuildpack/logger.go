package libbuildpack

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Logger interface {
	Info(format string, args ...interface{})
	Warning(format string, args ...interface{})
	Error(format string, args ...interface{})
	BeginStep(format string, args ...interface{})
	Protip(tip string, help_url string)
	GetOutput() io.Writer
	SetOutput(w io.Writer)
}

type logger struct {
	w io.Writer
}

func NewLogger() Logger {
	return &logger{w: os.Stdout}
}

func (l *logger) Info(format string, args ...interface{}) {
	l.printWithHeader("       ", format, args...)
}

func (l *logger) Warning(format string, args ...interface{}) {
	l.printWithHeader("       **WARNING** ", format, args...)

}
func (l *logger) Error(format string, args ...interface{}) {
	l.printWithHeader("       **ERROR** ", format, args...)
}

func (l *logger) BeginStep(format string, args ...interface{}) {
	l.printWithHeader("-----> ", format, args...)
}

func (l *logger) Protip(tip string, helpURL string) {
	l.printWithHeader("       PRO TIP: ", "%s", tip)
	l.printWithHeader("       Visit ", "%s", helpURL)
}

func (l *logger) printWithHeader(header string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)

	msg = strings.Replace(msg, "\n", "\n       ", -1)
	fmt.Fprintf(l.w, "%s%s\n", header, msg)
}

func (l *logger) GetOutput() io.Writer {
	return l.w
}

func (l *logger) SetOutput(w io.Writer) {
	l.w = w
}

var Log = &logger{w: os.Stdout}
