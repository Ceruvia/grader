package sandboxes

// SetOnUnmountBusy replaces the EBUSY hook until the returned restore runs.
func SetOnUnmountBusy(hook func()) (restore func()) {
	previous := onUnmountBusy
	onUnmountBusy = hook
	return func() { onUnmountBusy = previous }
}
