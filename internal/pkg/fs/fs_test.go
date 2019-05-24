package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

// The main criticism here is that we don't have to test os.Open, but we want 100% coverage.

func Test_OSFileSystem_EvalSymlinks(t *testing.T) {
	_, _ = OSFileSystem().EvalSymlinks("")
}

func Test_OSFileSystem_Open(t *testing.T) {
	file, err := OSFileSystem().Open("")
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err == nil {
		t.Fail()
	}
}

func Test_OSFileSystem_Stat(t *testing.T) {
	_, _ = OSFileSystem().Stat("")
}

func Test_MockFileSystem_Open_ENOENT(t *testing.T) {
	fs := MockFileSystem(map[string]MockFile{})
	file, err := fs.Open("")
	if file != nil {
		defer file.Close()
	}
	if !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_MockFileSystem(t *testing.T) {
	dataExpected := []byte("root:x:0:")
	fs := MockFileSystem(map[string]MockFile{
		"/passwd": {Content: dataExpected},
	})
	file, err := fs.Open("/passwd")
	if file != nil {
		defer file.Close()
	} else {
		t.Fail()
	}
	if err != nil {
		t.Error(err)
	}
	dataActual, err := ioutil.ReadAll(file)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(dataActual, dataExpected) {
		t.Fail()
	}
}


func Test_MockFileSystem_Stat(t *testing.T) {
	dataExpected := []byte("root:x:0:")
	fs := MockFileSystem(map[string]MockFile{
		"/passwd": {Content: dataExpected},
	})
	fileInfo, err := fs.Stat("/passwd")
	if err != nil {
		t.Error(err)
	}
	if fileInfo == nil || fileInfo.Name() != "passwd" {
		t.Fail()
	}
	if fileInfo == nil || fileInfo.Size() != 9 {
		t.Fail()
	}
	if fileInfo == nil || fileInfo.IsDir() {
		t.Fail()
	}
	fileInfo.Sys()
	fileInfo.ModTime()
	fileInfo.Mode()
}
