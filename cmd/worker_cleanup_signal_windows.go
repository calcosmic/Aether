//go:build windows

package cmd

func setupWorkerCleanupHandler() {
	// Worker process-group cleanup is Unix-only for now.
}
