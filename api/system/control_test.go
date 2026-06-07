package system

import "testing"

func TestRunServiceActionUnsupported(t *testing.T) {
	_, err := RunServiceAction("hostapd", "invalid")
	if err == nil {
		t.Fatal("expected unsupported action to return an error")
	}
}
