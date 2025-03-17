package logger

import (
	"bytes"
	"runtime"
	"strings"

	"github.com/docker/pinata/common/pkg/obfuscate"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/ztrue/tracerr"
)

const (
	// DateFormat is formatting used for date in logger, matches moby/moby see
	// https://github.com/moby/moby/blob/7b9275c0da707b030e62c96b679a976f31f929d3/pkg/jsonmessage/jsonmessage.go#L17
	DateFormat = "2006-01-02T15:04:05.000000000Z07:00"

	ShortDateFormat = "15:04:05.000000000Z07:00"

	// ComponentKey is used as a well known key when calling log.WithField(key, value)
	ComponentKey = "component"

	// BoldKey is used as a well known key when calling log.WithField(key, value)
	BoldKey = "bold"

	// StackMarker is used to append a multiline stack at the end of a log
	StackMarker = "[[STACK]]"
)

var (
	green     = color.New(color.FgGreen)
	red       = color.New(color.FgRed)
	boldGreen = color.New(color.FgWhite, color.BgGreen, color.Bold)
	boldRed   = color.New(color.FgWhite, color.BgRed, color.Bold)
)

func NewDesktopFormatter(name string, tty bool, dataFormat string) *desktopFormatter {
	return &desktopFormatter{
		name:       name,
		tty:        tty,
		dataFormat: dataFormat,
	}
}

type desktopFormatter struct {
	name       string
	tty        bool
	dataFormat string
}

var _ logrus.Formatter = &desktopFormatter{}

// Bytes returns the formatted log entry as a byte slice.
func (f *desktopFormatter) Bytes(e *logrus.Entry) []byte {
	var output bytes.Buffer
	output.Grow(60 + len(e.Message))

	// Date and time
	output.WriteByte('[')
	output.WriteString(e.Time.UTC().Format(f.dataFormat))
	output.WriteByte(']')

	// Component
	if len(f.name) > 0 {
		output.WriteByte('[')
		if f.tty {
			bold := e.Data[BoldKey] == true
			output.WriteString(colorFor(e.Level, bold).Sprintf("%-22s", label(e, f.name)))
		} else {
			output.WriteString(label(e, f.name))
		}
		output.WriteByte(']')
	}

	// Level
	output.WriteString(logrusLevel(e.Level))
	output.WriteByte(' ')

	// Message
	if f.tty {
		output.WriteString(e.Message)
		// On Windows, we need to add a carriage return before the newline character
		// to avoid the output from being ever indented to the right.
		if runtime.GOOS == "windows" {
			output.WriteByte('\r')
		}
		output.WriteByte('\n')
	} else {
		// Error
		if err, ok := e.Data[logrus.ErrorKey]; ok {
			if terr, ok := err.(tracerr.Error); ok {
				output.WriteString(e.Message)
				output.WriteString("\n")
				output.WriteString(sanitizeTracerrStack(terr))
				return output.Bytes()
			}
			if err, ok := err.(error); ok {
				output.WriteString(e.Message + ": " + err.Error())
				return output.Bytes()
			}
		}

		splitWithStackMarker := strings.SplitN(e.Message, StackMarker, 2)
		output.WriteString(obfuscate.ObfuscateString(splitWithStackMarker[0]))
		if !strings.HasSuffix(splitWithStackMarker[0], "\n") {
			output.WriteByte('\n')
		}
		if len(splitWithStackMarker) == 2 {
			output.WriteString(sanitizeStackFromMarker(splitWithStackMarker[1]))
		} else if e.Level == logrus.FatalLevel || e.Level == logrus.PanicLevel {
			if err, ok := e.Data[logrus.ErrorKey]; ok {
				if terr, ok := err.(stackTracer); ok {
					output.WriteString(e.Message)
					output.WriteByte('\n')
					output.WriteString(sanitizeErrorspkgStack(terr))
					return output.Bytes()
				}
			}
			output.WriteString(relevantStack())
		}
	}

	return output.Bytes()
}

func (f *desktopFormatter) Format(e *logrus.Entry) ([]byte, error) {
	return f.Bytes(e), nil
}

func label(e *logrus.Entry, name string) string {
	if component, ok := e.Data[ComponentKey]; ok {
		if componentName, ok := component.(string); ok {
			return name + "." + componentName
		}
	}
	return name
}

func logrusLevel(l logrus.Level) string {
	switch l {
	case logrus.PanicLevel:
		return "[P]"
	case logrus.FatalLevel:
		return "[F]"
	case logrus.ErrorLevel:
		return "[E]"
	case logrus.WarnLevel:
		return "[W]"
	case logrus.InfoLevel:
		return ""
	case logrus.DebugLevel:
		return "[D]"
	case logrus.TraceLevel:
		return "[T]"
	default:
		return ""
	}
}

func colorFor(l logrus.Level, isBold bool) *color.Color {
	if l <= logrus.ErrorLevel {
		if isBold {
			return boldRed
		}
		return red
	}
	if isBold {
		return boldGreen
	}
	return green
}
