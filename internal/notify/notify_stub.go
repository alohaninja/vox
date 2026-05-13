//go:build !darwin

package notify

import "fmt"

// StubNotifier prints notifications to stderr on non-darwin platforms.
type StubNotifier struct{}

// NewStubNotifier creates a stub notifier.
func NewStubNotifier() *StubNotifier { return &StubNotifier{} }

func (n *StubNotifier) Send(title, body string) {
	fmt.Printf("[notify] %s: %s\n", title, body)
}
