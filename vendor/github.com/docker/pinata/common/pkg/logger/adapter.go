package logger

import (
	"bufio"
	"io"
	"strings"
	"time"
)

// WriterAdapter ...
type WriterAdapter struct {
	Logger ComponentLogger
}

// Write ...
func (a *WriterAdapter) Write(p []byte) (int, error) {
	a.Logger.Print(string(p))
	return len(p), nil
}

type timestampWriter struct {
	pipeW io.Writer
}

// TimestampWriter prefixes timestamps onto each output line. This is useful when capturing
// stdout/stderr of processes which don't add timestamps of their own.
func TimestampWriter(w io.Writer) io.Writer {
	pipeR, pipeW := io.Pipe()

	br := bufio.NewReader(pipeR)
	go func() {
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				return
			}
			output := "[" + time.Now().UTC().Format(DateFormat) + "] " + string(line) + "\n"
			if _, err := io.Copy(w, strings.NewReader(output)); err != nil {
				return
			}
		}
	}()
	return &timestampWriter{
		pipeW: pipeW,
	}
}

func (w *timestampWriter) Write(p []byte) (int, error) {
	return w.pipeW.Write(p)
}
