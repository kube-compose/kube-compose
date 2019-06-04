package fs

import (
	"testing"
)

func TestNTVolumeNameLength_Case1(t *testing.T) {
	i := NTVolumeNameLength("")
	if i != 0 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case2(t *testing.T) {
	i := NTVolumeNameLength("///")
	if i != 0 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case3(t *testing.T) {
	i := NTVolumeNameLength("//")
	if i != 0 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case4(t *testing.T) {
	i := NTVolumeNameLength("//host//")
	if i != 0 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case5(t *testing.T) {
	i := NTVolumeNameLength("//host/servername")
	if i != 17 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case6(t *testing.T) {
	i := NTVolumeNameLength("//host/servername/path")
	if i != 17 {
		t.Fail()
	}
}

func TestNTVolumeNameLength_Case7(t *testing.T) {
	i := NTVolumeNameLength("C:\\Windows")
	if i != 2 {
		t.Fail()
	}
}
