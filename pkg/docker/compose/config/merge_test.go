package config

import (
	"reflect"
	"testing"
)

func Test_MergePortBindings_Basic(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func Test_MergePortBindings_Duplicate(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func Test_MergePortBindings_DuplicateInternalOnly(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8001, 8001, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8001, 8001, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func Test_MergePortBindings_Empty(t *testing.T) {
	intoPorts := []PortBinding{}
	fromPorts := []PortBinding{}
	expected := []PortBinding{}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}
func Test_MergeStringMaps_Basic(t *testing.T) {
	intoStringMap := map[string]string{"a": "b"}
	fromStringMap := map[string]string{"c": "d"}
	expected := map[string]string{"a": "b", "c": "d"}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}
func Test_MergeStringMaps_Duplicate(t *testing.T) {
	intoStringMap := map[string]string{"a": "b", "c": "d"}
	fromStringMap := map[string]string{"c": "d"}
	expected := map[string]string{"a": "b", "c": "d"}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}
func Test_MergeStringMaps_Empty(t *testing.T) {
	intoStringMap := map[string]string{}
	fromStringMap := map[string]string{}
	expected := map[string]string{}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}

func Test_Merge_Basic(t *testing.T) {
	serviceA := &serviceInternal{
		environmentParsed: map[string]string{"a": "b"},
		portsParsed:       []PortBinding{{80, 80, 80, "tcp", ""}},
	}

	serviceB := &serviceInternal{
		environmentParsed: map[string]string{"b": "c"},
		portsParsed:       []PortBinding{{8000, 8000, 8000, "tcp", ""}},
	}

	expected := &serviceInternal{
		environmentParsed: map[string]string{"a": "b", "b": "c"},
		portsParsed:       []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}},
	}

	merge(serviceA, serviceB, false)
	if !reflect.DeepEqual(serviceA, expected) {
		t.Fail()
	}
}
