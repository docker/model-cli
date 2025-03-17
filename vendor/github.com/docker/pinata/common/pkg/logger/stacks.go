package logger

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/ztrue/tracerr"
)

var pinataRoot = os.Getenv("PINATA_ROOT")

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func sanitizeTracerrStack(err tracerr.Error) string {
	stack := ""
	for _, v := range tracerr.StackTrace(err) {
		if strings.HasPrefix(v.Func, "runtime.") {
			break
		}
		pathWithLine := v.Path + ":" + fmt.Sprint(v.Line)
		stack += stackLineOutput(v.Func, pathWithLine)
	}
	return stack
}

func sanitizeErrorspkgStack(err stackTracer) string {
	stack := ""
	for _, frame := range err.StackTrace() {
		f := fmt.Sprintf("%+s", frame)
		if strings.HasPrefix(f, "runtime.") {
			break
		}
		f = strings.ReplaceAll(f, "\t", "")
		split := strings.SplitN(f, "\n", 2)
		pathWithLine := split[1] + ":" + fmt.Sprintf("%d", frame)
		stack += stackLineOutput(split[0], pathWithLine)
	}
	return stack
}

func stackLineOutput(fn, pathWithLine string) string {
	return "[" + fn + "()\n" +
		"[\t" + sanitizeStackPath(pathWithLine) + "\n" +
		codeSourceSection(pathWithLine)
}

func sanitizeStackFromMarker(rawstack string) string {
	stack := ""
	rawstack = strings.TrimSuffix(rawstack, "\"\n")
	rawstack = strings.ReplaceAll(rawstack, "\\n", "\n")
	scanner := bufio.NewScanner(strings.NewReader(rawstack))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.ReplaceAll(line, "\\t", "\t")
		line = strings.TrimSuffix(line, "\"\n")
		// add a [ to prevent stacks from memlogd to be prepended with host header
		stack += "[" + sanitizeStackPath(line) + "\n"
		stack += codeSourceSection(line)
	}
	return stack
}

func relevantStack() string {
	stack := ""
	scanner := bufio.NewScanner(strings.NewReader(string(debug.Stack())))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "[running]") &&
			!strings.Contains(line, "github.com/onsi/ginkgo/v2/internal.(*Suite)") &&
			!strings.Contains(line, "vendor/github.com/onsi/ginkgo/internal/suite") &&
			!strings.Contains(line, "logrus") &&
			!strings.Contains(line, "runtime/debug") &&
			!strings.Contains(line, "common/pkg/ipc") &&
			!strings.Contains(line, "common/pkg/logger") {
			// add a [ to prevent stacks from memlogd to be prepended with host header
			stack += "[" + sanitizeStackPath(line) + "\n"
			stack += codeSourceSection(line)
		}
	}
	return stack
}

func sanitizeStackPath(line string) string {
	pinataSplit := strings.SplitN(line, "/pinata/", 2)
	if len(pinataSplit) != 2 {
		return line
	}
	if pinataSplit[0][0] == '\t' {
		return "\t" + pinataSplit[1]
	}
	return pinataSplit[1]
}

func maxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func codeSourceSection(s string) string {
	if pinataRoot == "" {
		return ""
	}

	if cSharpIdx := strings.LastIndex(s, ") in"); cSharpIdx != -1 {
		s = strings.ReplaceAll(s[cSharpIdx+4:], ":line ", ":")
	}

	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return ""
	}
	file := s[:idx]
	line := s[idx+1:]
	if extraLineIdx := strings.LastIndex(line, " +0x"); extraLineIdx != -1 {
		line = line[:extraLineIdx]
	}

	l, err := strconv.Atoi(line)
	if err != nil {
		return ""
	}

	sourcePath := strings.TrimSpace(file)
	if i := strings.LastIndex(sourcePath, `\pinata\`); i != -1 {
		sourcePath = sourcePath[i+8:]
	}
	readFile, err := os.Open(filepath.Join(pinataRoot, sourcePath))
	if err != nil {
		readFile, err = os.Open(filepath.Join(pinataRoot, "vendor", sourcePath))
		if err != nil {
			return ""
		}
	}
	defer func() {
		_ = readFile.Close()
	}()
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	source := ""
	firstLine := maxInt(0, l-2)
	lastLine := l + 2
	currentLine := 0
	for fileScanner.Scan() {
		currentLine++
		if currentLine < firstLine {
			continue
		}
		if currentLine != l {
			source = source + "[+   " + fileScanner.Text() + "\n"
		} else {
			source = source + "[+-->" + fileScanner.Text() + "\n"
		}
		if currentLine == lastLine {
			break
		}
	}
	return "[\n" + source + "[------------------\n"
}
