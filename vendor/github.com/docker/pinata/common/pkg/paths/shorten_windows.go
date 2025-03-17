package paths

func ShortenUnixSocketPath(path string) (string, error) {
	// Nothing to do on Windows
	return path, nil
}
