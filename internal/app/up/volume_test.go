package up

import (
	"archive/tar"
	"os"
	"reflect"
	"testing"

	fsPackage "github.com/jbrekelmans/kube-compose/internal/pkg/fs"
)

var mockFileSystem = fsPackage.MockFileSystem(map[string]fsPackage.MockFile{
	"orig": {
		Content: []byte("content"),
	},
})

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
			Mode:     int64(os.ModePerm),
			Size:     int64(len(data)),
		},
		data: []byte(data),
	}
}

func TestBindMouseHostFileToTar_Success(t *testing.T) {
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
				regularFile("renamed", "content"),
			}
			if !reflect.DeepEqual(tw.entries, expected) {
				t.Logf("entries1: %+v\n", tw.entries)
				t.Logf("entries2: %+v\n", expected)
				t.Fail()
			}
		}
	})
}
