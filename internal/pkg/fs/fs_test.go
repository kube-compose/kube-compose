package fs

import (
	"bytes"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/pkg/errors"
)

// The main criticism here is that we don't have to test os.Open, but we want 100% coverage.

func Test_OSFileSystem_EvalSymlinks(t *testing.T) {
	_, _ = OSFileSystem().EvalSymlinks("")
}

func Test_OSFileSystem_Lstat(t *testing.T) {
	_, _ = OSFileSystem().Lstat("")
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
	fs := NewMockFileSystem(map[string]MockFile{})
	file, err := fs.Open("/data")
	if file != nil {
		defer file.Close()
	}
	if !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_MockFileSystem(t *testing.T) {
	dataExpected := []byte("root:x:0:")
	fs := NewMockFileSystem(map[string]MockFile{
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

func Test_MockFileSystem_SuccessEmptyString(t *testing.T) {
	NewMockFileSystem(map[string]MockFile{
		"": {
			Mode: os.ModeDir,
		},
	})
}

func Test_MockFileSystem_InvalidMode1(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	NewMockFileSystem(map[string]MockFile{
		"/invalidmode1": {
			Mode: os.ModeDir | os.ModeSymlink,
		},
	})
}

func Test_MockFileSystem_InvalidMode2(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	NewMockFileSystem(map[string]MockFile{
		"/invalidmode2": {
			Mode: os.ModeDevice | os.ModeSymlink,
		},
	})
}

func Test_MockFileSystem_DirectoryInconsistency1(t *testing.T) {
	defer func() {
		err := recover()
		if err != errMockDirectoryInconsistency {
			t.Fail()
		}
	}()
	var fs = NewMockFileSystem(map[string]MockFile{
		"/dir/fileforreal": {
			Content: []byte("regularfile"),
		},
	}).(*mockFileSystem)
	fs.Set("/dir", MockFile{
		Content: []byte("notafile"),
	})
}
func Test_MockFileSystem_DirectoryInconsistency2(t *testing.T) {
	defer func() {
		err := recover()
		if err != errMockDirectoryInconsistency {
			t.Fail()
		}
	}()
	var fs = NewMockFileSystem(map[string]MockFile{
		"/dir": {
			Content: []byte("regularfile2"),
		},
	}).(*mockFileSystem)
	fs.Set("/dir/fileforreal2", MockFile{
		Content: []byte("regularfile3"),
	})
}

func Test_MockFileDescriptor_Read_EmptyBuffer(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{
		"/emptybuffer": {Content: []byte("nope")},
	})
	fd, err := fs.Open("/emptybuffer")
	if err != nil {
		t.Error(err)
	} else {
		n, err := fd.Read(nil)
		if err != nil {
			t.Error(err)
		}
		if n != 0 {
			t.Fail()
		}
	}
}
func Test_MockFileDescriptor_Readdir_ENOTDIR(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{
		"/enotdir": {Content: []byte("ENOTDIR")},
	})
	fd, err := fs.Open("/enotdir")
	if err != nil {
		t.Error(err)
	} else {
		_, err = fd.Readdir(0)
		if err != syscall.ENOTDIR {
			t.Fail()
		}
	}
}
func Test_MockFileDescriptor_Readdir_NNotSupported(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{})
	fd, err := fs.Open("")
	if err != nil {
		t.Error(err)
	} else {
		defer func() {
			err := recover()
			if err == nil {
				t.Fail()
			}
		}()
		_, err = fd.Readdir(3)
		if err != nil {
			t.Error(err)
		}
	}
}

func Test_MockFileSystem_Lstat_Success(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{
		"/passwd":      {Content: []byte("root:x:0:")},
		"/path/to/dir": {Mode: os.ModeDir},
	})
	fileInfo, err := fs.Lstat("/passwd")
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

func Test_MockFileSystem_Lstat_InvalidPath(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	fs := NewMockFileSystem(map[string]MockFile{})
	_, _ = fs.Lstat("/.")
}

func Test_MockFileSystem_Set_ReplacesFileContentsCorrectly(t *testing.T) {
	name := "/replacesfilecontents"
	fs := NewMockFileSystem(map[string]MockFile{
		name: {Content: []byte("filecontentsorig")},
	})
	expected := []byte("filecontentsreplaces")
	fs.Set(name, MockFile{
		Content: expected,
	})
	fd, err := fs.Open(name)
	if err != nil {
		t.Error(err)
	} else {
		actual, err := ioutil.ReadAll(fd)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fail()
		}
	}
}

func Test_MockFileSystem_Stat_ENOENT(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{})
	_, err := fs.Stat("/passwd")
	if err == nil || !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_DirError2(t *testing.T) {
	errExpected := errors.New("unknown error 14")
	fs := NewMockFileSystem(map[string]MockFile{
		"/": {
			Error: errExpected,
			Mode:  os.ModeDir,
		},
	})
	_, errActual := fs.Stat("/")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_DirError3(t *testing.T) {
	errExpected := errors.New("unknown error 15")
	fs := NewMockFileSystem(map[string]MockFile{
		"/": {
			Error: errExpected,
			Mode:  os.ModeDir,
		},
	})
	_, errActual := fs.Stat("/asdf")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_FileError(t *testing.T) {
	errExpected := errors.New("unknown error 12")
	fs := NewMockFileSystem(map[string]MockFile{
		"/passwd": {Error: errExpected},
	})
	_, errActual := fs.Stat("/passwd")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_ENOTDIR(t *testing.T) {
	fs := NewMockFileSystem(map[string]MockFile{
		"/enotdir2": {},
	})
	_, err := fs.Stat("/enotdir2/file3")
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}
