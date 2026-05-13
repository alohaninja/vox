//go:build darwin

package notify

import "testing"

func TestNotifierInterface(t *testing.T) {
	// Verify both implementations satisfy the interface.
	var _ Notifier = (*DarwinNotifier)(nil)
}

func TestSendSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping notification test in short mode")
	}
	n := NewDarwinNotifier()
	n.Send("Vox Test", "This is a test notification from vox.")
}
