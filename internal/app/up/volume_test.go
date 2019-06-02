package up

import (
	"archive/tar"
	"bytes"
	"fmt"
	"reflect"
	"testing"

	fsPackage "github.com/jbrekelmans/kube-compose/internal/pkg/fs"
)

var errTest = fmt.Errorf("test error")
var testFileContent = "content"

var mockFileSystem fsPackage.MockFileSystem

// Init here is justified because a common mock file system is used, and we require calling Add to make tests deterministic.
// nolint
func init() {
	mockFileSystem = fsPackage.NewMockFileSystem(map[string]fsPackage.MockFile{
		"/orig": {
			Content: []byte(testFileContent),
		},
		"/origerr": {
			Error: errTest,
		},
	})
	mockFileSystem.Add("/dir/file1", fsPackage.MockFile{
		Content: []byte(testFileContent),
	})
	mockFileSystem.Add("/dir/file2", fsPackage.MockFile{
		Content: []byte(testFileContent),
	})
}

func withMockFS(cb func()) {
	fsOld := fs
	defer func() {
		fs = fsOld
	}()
	fs = mockFileSystem
	cb()
}

type mockTarWriterEntry struct {
	h    *tar.Header
	data []byte
}

type mockTarWriter struct {
	entries []mockTarWriterEntry
}

func (m *mockTarWriter) WriteHeader(header *tar.Header) error {
	m.entries = append(m.entries, mockTarWriterEntry{
		h: header,
	})
	return nil
}

func (m *mockTarWriter) Write(p []byte) (int, error) {
	entry := &m.entries[len(m.entries)-1]
	entry.data = append(entry.data, p...)
	return len(p), nil
}

func regularFile(name, data string) mockTarWriterEntry {
	return mockTarWriterEntry{
		h: &tar.Header{
			Name:     name,
			Typeflag: tar.TypeReg,
			Size:     int64(len(data)),
		},
		data: []byte(data),
	}
}

func directory(name string) mockTarWriterEntry {
	return mockTarWriterEntry{
		h: &tar.Header{
			Name:     name,
			Typeflag: tar.TypeDir,
		},
	}
}

func TestBindMouseHostFileToTar_SuccessRegularFile(t *testing.T) {
	withMockFS(func() {
		tw := &mockTarWriter{}
		isDir, err := bindMouseHostFileToTar(tw, "orig", "renamed")
		if err != nil {
			t.Error(err)
		} else {
			if isDir {
				t.Fail()
			}
			expected := []mockTarWriterEntry{
				regularFile("renamed", testFileContent),
			}
			if !reflect.DeepEqual(tw.entries, expected) {
				t.Logf("entries1: %+v\n", tw.entries)
				t.Logf("entries2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}

func TestBindMouseHostFileToTar_RecoverFromRegularFileError(t *testing.T) {
	withMockFS(func() {
		tw := &mockTarWriter{}
		isDir, err := bindMouseHostFileToTar(tw, "origerr", "renamed")
		if err != nil {
			t.Error(err)
		} else {
			if !isDir {
				t.Fail()
			}
			expected := []mockTarWriterEntry{
				directory("renamed/"),
			}
			if !reflect.DeepEqual(tw.entries, expected) {
				t.Logf("entries1: %+v\n", tw.entries)
				t.Logf("entries2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}

func TestBindMouseHostFileToTar_SuccessDir(t *testing.T) {
	withMockFS(func() {
		tw := &mockTarWriter{}
		isDir, err := bindMouseHostFileToTar(tw, "dir", "renamed")
		if err != nil {
			t.Error(err)
		} else {
			if !isDir {
				t.Fail()
			}
			expected := []mockTarWriterEntry{
				directory("renamed/"),
				regularFile("renamed/file1", testFileContent),
				regularFile("renamed/file2", testFileContent),
			}
			if !reflect.DeepEqual(tw.entries, expected) {
				t.Logf("entries1: %+v\n", tw.entries)
				t.Logf("entries2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}

func TestBuildVolumeInitImageGetDockerfile_Success(t *testing.T) {
	actual := buildVolumeInitImageGetDockerfile([]bool{true, false})
	expected := []byte(`ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY data1/ /app/data/vol1/
COPY data2 /app/data/vol2
ENTRYPOINT ["bash", "-c", "cp -ar /app/data/vol1 /mnt/vol1/root && cp -ar /app/data/vol2 /mnt/vol2/root"]
`)
	if !bytes.Equal(actual, expected) {
		t.Logf("actual:\n%s", string(actual))
		t.Logf("expected:\n%s", string(expected))
		t.Fail()
	}
}
