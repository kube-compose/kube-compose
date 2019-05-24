package up

import (
	"testing"
)

func TestPodStatusString_Ready(t *testing.T) {
	status := podStatusReady
	str := (&status).String()
	if str != podStatusReadyString {
		t.Fail()
	}
}

func TestPodStatusString_Started(t *testing.T) {
	status := podStatusStarted
	str := (&status).String()
	if str != podStatusStartedString {
		t.Fail()
	}
}

func TestPodStatusString_Completed(t *testing.T) {
	status := podStatusCompleted
	str := (&status).String()
	if str != podStatusCompletedString {
		t.Fail()
	}
}

func TestPodStatusString_Other(t *testing.T) {
	status := podStatus(-1)
	str := (&status).String()
	if str != podStatusOtherString {
		t.Fail()
	}
}
