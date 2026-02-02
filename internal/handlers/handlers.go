package handlers

import (
	"golinks/internal/email"
)

// Notifier is the global email notifier instance.
// Set during application initialization.
var Notifier *email.Notifier

// SetNotifier sets the global email notifier.
func SetNotifier(n *email.Notifier) {
	Notifier = n
}
