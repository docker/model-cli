package standalone

// StatusPrinter is the interface used to print status updates.
type StatusPrinter interface {
	// Printf should perform formatted printing.
	Printf(format string, args ...any)
	// Println should perform line-based printing.
	Println(args ...any)
}
