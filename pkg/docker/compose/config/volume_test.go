package config

import (
	"reflect"
	"testing"

	"github.com/kube-compose/kube-compose/pkg/expanduser"
)

func TestParsePathMapping_Case1(t *testing.T) {
	r := parsePathMapping("aa")
	if !reflect.DeepEqual(r, PathMapping{
		ContainerPath: "aa",
	}) {
		t.Fail()
	}
}
func TestParsePathMapping_Case2(t *testing.T) {
	r := parsePathMapping("aa:bb")
	if !reflect.DeepEqual(r, PathMapping{
		HasHostPath:   true,
		HostPath:      "aa",
		ContainerPath: "bb",
	}) {
		t.Fail()
	}
}
func TestParsePathMapping_Case3(t *testing.T) {
	r := parsePathMapping("aa:bb:cc")
	if !reflect.DeepEqual(r, PathMapping{
		ContainerPath: "bb",
		HasHostPath:   true,
		HasMode:       true,
		HostPath:      "aa",
		Mode:          "cc",
	}) {
		t.Logf("pathMapping: %+v\n", r)
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case1(t *testing.T) {
	i := ntpathVolumeNameLength("")
	if i != 0 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case2(t *testing.T) {
	i := ntpathVolumeNameLength("///")
	if i != 0 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case3(t *testing.T) {
	i := ntpathVolumeNameLength("//")
	if i != 0 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case4(t *testing.T) {
	i := ntpathVolumeNameLength("//host//")
	if i != 0 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case5(t *testing.T) {
	i := ntpathVolumeNameLength("//host/servername")
	if i != 17 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case6(t *testing.T) {
	i := ntpathVolumeNameLength("//host/servername/path")
	if i != 17 {
		t.Fail()
	}
}

func TestNTPathVolumeNameLength_Case7(t *testing.T) {
	i := ntpathVolumeNameLength("C:\\Windows")
	if i != 2 {
		t.Fail()
	}
}

func TestVolumeNameLength_Case1(t *testing.T) {
	i := volumeNameLength("")
	if i != 0 {
		t.Fail()
	}
}
func TestVolumeNameLength_Case2(t *testing.T) {
	i := volumeNameLength(".")
	if i != 0 {
		t.Fail()
	}
}
func TestVolumeNameLength_Case3(t *testing.T) {
	i := volumeNameLength("\\")
	if i != 0 {
		t.Fail()
	}
}
func TestVolumeNameLength_Case4(t *testing.T) {
	i := volumeNameLength("/")
	if i != 0 {
		t.Fail()
	}
}
func TestVolumeNameLength_Case5(t *testing.T) {
	i := volumeNameLength("~")
	if i != 0 {
		t.Fail()
	}
}
func TestVolumeNameLength_Case6(t *testing.T) {
	i := volumeNameLength("C:\\Users\\henk")
	if i != 2 {
		t.Fail()
	}
}

func TestResolveBindMountVolumeHostPath_Success(t *testing.T) {
	sv := ServiceVolume{
		Short: &PathMapping{
			HasHostPath: true,
			HostPath:    "./Documents",
		},
	}
	resolveBindMountVolumeHostPath("/Users/henk/.bash_profile", &sv)
	expected := ServiceVolume{
		Short: &PathMapping{
			HasHostPath: true,
			HostPath:    "/Users/henk/Documents",
		},
	}
	if !reflect.DeepEqual(sv, expected) {
		t.Logf("serviceVolume1: %+v\n", sv)
		t.Logf("serviceVolume2: %+v\n", expected)
		t.Fail()
	}
}
func TestResolveBindMountVolumeHostPath_TildeNotSupported(t *testing.T) {
	sv := ServiceVolume{
		Short: &PathMapping{
			HasHostPath: true,
			HostPath:    "~/Documents",
		},
	}
	expanduser.LookupEnvFunc = func(name string) (string, bool) {
		if name == "HOME" || name == "USERPROFILE" {
			return "/home/henk", true
		}
		return "", false
	}
	resolveBindMountVolumeHostPath("/Users/henk/.bash_profile", &sv)
}
