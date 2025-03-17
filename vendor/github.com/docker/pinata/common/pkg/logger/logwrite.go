package logger

// Write VM logs to files and perform rotation.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

const mb = 1024 * 1024

const (
	maxLogFiles = 10
	maxLogSize  = mb
)

// LogFile is where we write LogMessages to
type LogFile struct {
	File         *os.File // active file handle
	Path         string   // Path to the logfile
	bytesWritten int      // total number of bytes written so far
	MaxLogFiles  int      // MaxLogFiles
	key          string   // key in logs map
	m            sync.Mutex
}

var _ io.Writer = &LogFile{}

var (
	logs  = make(map[string]*LogFile)
	logsM sync.Mutex
)

// Open a logfile, creating it if it doesn't exist.
func Open(dir, name string) (*LogFile, error) {
	key := filepath.Join(dir, name)
	logsM.Lock()
	defer logsM.Unlock()
	logF, ok := logs[key]
	if ok {
		return logF, nil
	}
	logF, err := newLogFile(key, dir, name)
	if err != nil {
		return nil, err
	}
	if err := writeOpenBanner(logF); err != nil {
		return nil, err
	}
	logs[key] = logF
	return logF, nil
}

// Make every new logging session obvious when searching through the file.
func writeOpenBanner(w io.Writer) error {
	if _, err := w.Write([]byte("-------------------------------------------------------------------------------->8\n")); err != nil {
		return err
	}
	return nil
}

func newLogFile(key, dir, name string) (*LogFile, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrap(err, "creating "+dir)
	}
	// If the log exists already we want to append to it.
	p := filepath.Join(dir, name+".log")
	f, err := openFile(p)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return &LogFile{
		File:         f,
		Path:         p,
		bytesWritten: int(fi.Size()),
		MaxLogFiles:  maxLogFiles,
		key:          key,
	}, nil
}

func (l *LogFile) Write(b []byte) (int, error) {
	l.m.Lock()
	defer l.m.Unlock()
	if l.File == nil {
		// This Write() has been called after Close()
		return 0, io.EOF
	}
	// If this write would push us over the limit, rotate first.
	if l.bytesWritten+len(b) > maxLogSize {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := l.File.Write(b)
	l.bytesWritten += n
	return n, err
}

// Close a log file. It can be re-opened later.
func (l *LogFile) Close() error {
	l.m.Lock()
	err := l.File.Close()
	l.File = nil
	l.m.Unlock()

	// There shouldn't be calls to Write() after Close() but if there are, they will see that File is nil.

	logsM.Lock()
	delete(logs, l.key)
	logsM.Unlock()
	return err
}

func (l *LogFile) rotate() error {
	// Called with the mutex already locked. The file may be open or closed.
	if l.File != nil {
		if err := l.File.Close(); err != nil {
			return err
		}
	}
	for i := l.MaxLogFiles - 2; i >= 0; i-- {
		newerFile := fmt.Sprintf("%s.%d", l.Path, i-1)
		// special case: if index is 0 we omit the suffix i.e. we expect
		// foo foo.1 foo.2 up to foo.<maxLogFiles-2>
		if i == 0 {
			newerFile = l.Path
		}
		olderFile := fmt.Sprintf("%s.%d", l.Path, i)
		// overwrite the olderFile with the newerFile
		err := os.Rename(newerFile, olderFile)
		if os.IsNotExist(err) {
			// the newerFile does not exist
			continue
		}
		if err != nil {
			return err
		}
	}
	if l.File != nil {
		f, err := os.Create(l.Path)
		if err != nil {
			return err
		}
		l.File = f
		l.bytesWritten = 0
	}
	return nil
}

// Rotate closes the current log file, rotates the files and creates an empty log file.
func (l *LogFile) Rotate() error {
	l.m.Lock()
	defer l.m.Unlock()
	return l.rotate()
}
