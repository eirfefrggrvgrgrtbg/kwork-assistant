package email

import (
	"fmt"
	"testing"
)

func TestMessageIDFallback(t *testing.T) {
	// Simulated email A
	msgUidA := uint32(100)
	hashA := "AAA"
	msgIdA := ""
	if msgIdA == "" {
		msgIdA = fmt.Sprintf("UID-%d-%s", msgUidA, hashA)
	}

	// Simulated email B
	msgUidB := uint32(101)
	hashB := "BBB"
	msgIdB := ""
	if msgIdB == "" {
		msgIdB = fmt.Sprintf("UID-%d-%s", msgUidB, hashB)
	}

	if msgIdA == msgIdB {
		t.Errorf("Expected different fallback message IDs, got the same: %s", msgIdA)
	}

	expectedA := "UID-100-AAA"
	if msgIdA != expectedA {
		t.Errorf("Expected %s, got %s", expectedA, msgIdA)
	}
}
