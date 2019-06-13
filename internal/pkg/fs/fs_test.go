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
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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
	NewVirtualFileSystem(map[string]VirtualFile{
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
	NewVirtualFileSystem(map[string]VirtualFile{
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
	NewVirtualFileSystem(map[string]VirtualFile{
		"/invalidmode2": {
			Mode: os.ModeDevice | os.ModeSymlink,
		},
	})
}

func Test_MockFileSystem_DirectoryInconsistency1(t *testing.T) {
	defer func() {
		err := recover()
		if err != errIsDirDisagreement {
			t.Fail()
		}
	}()
	var fs = NewVirtualFileSystem(map[string]VirtualFile{
		"/dir/fileforreal": {
			Content: []byte("regularfile"),
		},
	}).(*virtualFileSystem)
	fs.Set("/dir", VirtualFile{
		Content: []byte("notafile"),
	})
}
func Test_MockFileSystem_DirectoryInconsistency2(t *testing.T) {
	defer func() {
		err := recover()
		if err != errIsDirDisagreement {
			t.Fail()
		}
	}()
	var fs = NewVirtualFileSystem(map[string]VirtualFile{
		"/dir": {
			Content: []byte("regularfile2"),
		},
	}).(*virtualFileSystem)
	fs.Set("/dir/fileforreal2", VirtualFile{
		Content: []byte("regularfile3"),
	})
}

func Test_MockFileDescriptor_Read_EmptyBuffer(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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

func Test_MockFileDescriptor_Read_EISDIR(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	fd, err := fs.Open("/")
	if err != nil {
		t.Error(err)
	} else {
		_, err = fd.Read(nil)
		if err != errBadMode {
			t.Error(err)
		}
	}
}

func Test_MockFileDescriptor_Readdir_ENOTDIR(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	_, _ = fs.Lstat("/.")
}

func Test_MockFileSystem_Set_ReplacesFileContentsCorrectly(t *testing.T) {
	name := "/replacesfilecontents"
	fs := NewVirtualFileSystem(map[string]VirtualFile{
		name: {Content: []byte("filecontentsorig")},
	})
	expected := []byte("filecontentsreplaces")
	fs.Set(name, VirtualFile{
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{})
	_, err := fs.Stat("/passwd")
	if err == nil || !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_DirError2(t *testing.T) {
	errExpected := errors.New("unknown error 14")
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{
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
	fs := NewVirtualFileSystem(map[string]VirtualFile{
		"/passwd": {Error: errExpected},
	})
	_, errActual := fs.Stat("/passwd")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_MockFileSystem_Stat_ENOTDIR(t *testing.T) {
	fs := NewVirtualFileSystem(map[string]VirtualFile{
		"/enotdir2": {},
	})
	_, err := fs.Stat("/enotdir2/file3")
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}
