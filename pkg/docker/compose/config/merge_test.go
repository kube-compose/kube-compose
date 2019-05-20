package config

import (
	"reflect"
	"testing"
)

func TestMergePortBindings_Basic(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func TestMergePortBindings_Duplicate(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func TestMergePortBindings_DuplicateInternalOnly(t *testing.T) {
	intoPorts := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8001, 8001, "tcp", ""}}
	fromPorts := []PortBinding{{8000, 8000, 8000, "tcp", ""}}
	expected := []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8001, 8001, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}

func TestMergePortBindings_Empty(t *testing.T) {
	intoPorts := []PortBinding{}
	fromPorts := []PortBinding{}
	expected := []PortBinding{}

	intoPorts = mergePortBindings(intoPorts, fromPorts)
	if !reflect.DeepEqual(intoPorts, expected) {
		t.Fail()
	}
}
func TestMergeStringMaps_Basic(t *testing.T) {
	intoStringMap := map[string]string{"a": "b"}
	fromStringMap := map[string]string{"c": "d"}
	expected := map[string]string{"a": "b", "c": "d"}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}
func TestMergeStringMaps_Duplicate(t *testing.T) {
	intoStringMap := map[string]string{"a": "b", "c": "d"}
	fromStringMap := map[string]string{"c": "d"}
	expected := map[string]string{"a": "b", "c": "d"}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}
func TestMergeStringMaps_Empty(t *testing.T) {
	intoStringMap := map[string]string{}
	fromStringMap := map[string]string{}
	expected := map[string]string{}

	mergeStringMaps(intoStringMap, fromStringMap)
	if !reflect.DeepEqual(intoStringMap, expected) {
		t.Fail()
	}
}

func TestMerge_Basic(t *testing.T) {
	serviceA := &composeFileParsedService{
		service: &Service{
			Environment: map[string]string{"a": "b"},
			Ports:       []PortBinding{{80, 80, 80, "tcp", ""}},
		},
	}

	serviceB := &composeFileParsedService{
		service: &Service{
			Environment: map[string]string{"b": "c"},
			Ports:       []PortBinding{{8000, 8000, 8000, "tcp", ""}},
		},
	}

	expected := &composeFileParsedService{
		service: &Service{
			Environment: map[string]string{"a": "b", "b": "c"},
			Ports:       []PortBinding{{80, 80, 80, "tcp", ""}, {8000, 8000, 8000, "tcp", ""}},
		},
	}

	merge(serviceA, serviceB)
	if !reflect.DeepEqual(serviceA, expected) {
		t.Fail()
	}
}
