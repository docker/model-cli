package distribution

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// ProgressMessage represents a structured message for progress reporting
type ProgressMessage struct {
	Type    string `json:"type"`    // "progress", "success", or "error"
	Message string `json:"message"` // Human-readable message
	Total   uint64 `json:"total"`   // Total bytes to transfer
	Pulled  uint64 `json:"pulled"`  // Bytes transferred so far
}

type reporter struct {
	progress chan v1.Update
	done     chan struct{}
	err      error
	out      io.Writer
	format   progressF
}

type progressF func(update v1.Update) string

func pullMsg(update v1.Update) string {
	return fmt.Sprintf("Downloaded: %.2f MB", float64(update.Complete)/1024/1024)
}

func pushMsg(update v1.Update) string {
	return fmt.Sprintf("Uploaded: %.2f MB", float64(update.Complete)/1024/1024)
}

func newProgressReporter(w io.Writer, msgF progressF) *reporter {
	return &reporter{
		out:      w,
		progress: make(chan v1.Update),
		done:     make(chan struct{}),
		format:   msgF,
	}
}

// safeUint64 converts an int64 to uint64, ensuring the value is non-negative
func safeUint64(n int64) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

// updates returns a channel for receiving progress updates. It is the responsibility of the caller to close
// the channel when they are done sending updates. Should only be called once per reporter instance.
func (r *reporter) updates() chan<- v1.Update {
	go func() {
		var lastComplete int64
		var lastUpdate time.Time
		const updateInterval = 500 * time.Millisecond // Update every 500ms
		const minBytesForUpdate = 1024 * 1024         // At least 1MB difference

		for p := range r.progress {
			if r.out == nil || r.err != nil {
				continue // If we fail to write progress, don't try again
			}
			now := time.Now()
			bytesDownloaded := p.Complete - lastComplete
			// Only update if enough time has passed or enough bytes downloaded or finished
			if now.Sub(lastUpdate) >= updateInterval ||
				bytesDownloaded >= minBytesForUpdate {
				if err := writeProgress(r.out, r.format(p), safeUint64(p.Total), safeUint64(p.Complete)); err != nil {
					r.err = err
				}
				lastUpdate = now
				lastComplete = p.Complete
			}
		}
		close(r.done) // Close the done channel when progress is complete
	}()
	return r.progress
}

// Wait waits for the progress reporter to finish and returns any error encountered.
func (r *reporter) Wait() error {
	<-r.done
	return r.err
}

// writeProgressMessage writes a JSON-formatted progress message to the writer
func writeProgressMessage(w io.Writer, msg ProgressMessage) error {
	if w == nil {
		return nil
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

// writeProgress writes a progress update message
func writeProgress(w io.Writer, msg string, total, pulled uint64) error {
	return writeProgressMessage(w, ProgressMessage{
		Type:    "progress",
		Message: msg,
		Total:   total,
		Pulled:  pulled,
	})
}

// writeSuccess writes a success message
func writeSuccess(w io.Writer, message string) error {
	return writeProgressMessage(w, ProgressMessage{
		Type:    "success",
		Message: message,
	})
}

// writeError writes an error message
func writeError(w io.Writer, message string) error {
	return writeProgressMessage(w, ProgressMessage{
		Type:    "error",
		Message: message,
	})
}
