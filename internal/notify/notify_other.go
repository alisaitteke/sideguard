//go:build !darwin && !linux

package notify

import "fmt"

func sendMacOS(title, body string) error {
	return fmt.Errorf("macOS notifications are not supported on %s", "non-darwin")
}
