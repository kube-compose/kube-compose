package fs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/pkg/errors"
)

// We don't have to test "os" and "path/filepath" functions, but we want 100% coverage.

func Test_OSFileSystem_Abs(t *testing.T) {
	_, _ = OS.Abs("")
}

func Test_OSFileSystem_Getwd(t *testing.T) {
	_, _ = OS.Getwd()
}

func Test_OSFileSystem_Lstat(t *testing.T) {
	_, _ = OS.Lstat("")
}

func Test_OSFileSystem_Mkdir(t *testing.T) {
	_ = OS.Mkdir("", os.ModePerm)
}

func Test_OSFileSystem_MkdirAll(t *testing.T) {
	_ = OS.MkdirAll("", os.ModePerm)
}

func Test_OSFileSystem_Open(t *testing.T) {
	file, err := OS.Open("")
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err == nil {
		t.Fail()
	}
}
func Test_OSFileSystem_Readlink(t *testing.T) {
	_, _ = OS.Readlink("")
}

func Test_OSFileSystem_Stat(t *testing.T) {
	_, _ = OS.Stat("")
}

func Test_VirtualFileSystem_Abs_InjectedFault(t *testing.T) {
	errExpected := fmt.Errorf("absInjectedFault")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	fs.AbsError = errExpected
	_, errActual := fs.Abs("")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Open_ENOENT(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	file, err := fs.Open("/data")
	if file != nil {
		defer file.Close()
	}
	if !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Open_OpenError(t *testing.T) {
	errExpected := fmt.Errorf("openError")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/openerror": {
			OpenError: errExpected,
		},
	})
	_, errActual := fs.Open("/openerror")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem(t *testing.T) {
	dataExpected := []byte("root:x:0:")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
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

func Test_VirtualFileSystem_SuccessEmptyString(t *testing.T) {
	NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"": {
			Mode: os.ModeDir,
		},
	})
}

func Test_VirtualFileSystem_InvalidMode1(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/invalidmode1": {
			Mode: os.ModeDir | os.ModeSymlink,
		},
	})
}

func Test_VirtualFileSystem_InvalidMode2(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/invalidmode2": {
			Mode: os.ModeDevice | os.ModeSymlink,
		},
	})
}

func Test_VirtualFileSystem_DirectoryInconsistency1(t *testing.T) {
	defer func() {
		err := recover()
		if err != errIsDirDisagreement {
			t.Fail()
		}
	}()
	var fs = NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/dir/fileforreal": {
			Content: []byte("regularfile"),
		},
	})
	fs.Set("/dir", &InMemoryFile{
		Content: []byte("notafile"),
	})
}

func Test_VirtualFileSystem_DirectoryInconsistency2(t *testing.T) {
	defer func() {
		err := recover()
		if err != errIsDirDisagreement {
			t.Fail()
		}
	}()
	var fs = NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/dir": {
			Content: []byte("regularfile2"),
		},
	})
	fs.Set("/dir/fileforreal2", &InMemoryFile{
		Content: []byte("regularfile3"),
	})
}

func Test_VirtualFileSystem_GetwdError(t *testing.T) {
	var fs = NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	errExpected := fmt.Errorf("getwderror")
	fs.GetwdError = errExpected
	_, errActual := fs.Getwd()
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_GetwdSuccess(t *testing.T) {
	var fs = NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	cwd, err := fs.Getwd()
	if err != nil {
		t.Error(err)
	} else if cwd != "/" {
		t.Fail()
	}
}

func Test_VirtualFileDescriptor_Read_EmptyBuffer(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
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

func Test_VirtualFileDescriptor_Read_EISDIR(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
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

func Test_VirtualFileDescriptor_Read_ReadError(t *testing.T) {
	errExpected := fmt.Errorf("readError")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/readerror": {
			Content:   []byte("readerrorcontent"),
			ReadError: errExpected,
		},
	})
	fd, err := fs.Open("/readerror")
	defer func() {
		if fd != nil {
			fd.Close()
		}
	}()
	if err != nil {
		t.Error(err)
	} else {
		_, errActual := fd.Read([]byte{})
		if errActual != errExpected {
			t.Fail()
		}
	}
}

func Test_VirtualFileDescriptor_Readdir_ENOTDIR(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
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

func Test_VirtualFileDescriptor_Readdir_ReadError(t *testing.T) {
	errExpected := fmt.Errorf("readdirError")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/": {
			Mode:      os.ModeDir,
			ReadError: errExpected,
		},
	})
	fd, err := fs.Open("/")
	defer func() {
		if fd != nil {
			fd.Close()
		}
	}()
	if err != nil {
		t.Error(err)
	} else {
		_, errActual := fd.Readdir(0)
		if errActual != errExpected {
			t.Fail()
		}
	}
}

func Test_VirtualFileSystem_Readdir_EmptyDirSuccess(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		// Trailing slash is intentional.
		"/dir1/dir1_1/": {},
	})
	fd, err := fs.Open("/dir1/dir1_1")
	if err != nil {
		t.Error(err)
	} else {
		dir, err := fd.Readdir(0)
		if err != nil {
			t.Error(err)
		} else if len(dir) != 0 {
			t.Fail()
		}
	}
}

func Test_VirtualFileDescriptor_Readdir_NNotSupported(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
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

func Test_VirtualFileSystem_Lstat_Success(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/passwd":       {Content: []byte("root:x:0:")},
		"/path/to/dir/": {Mode: os.ModeDir},
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

func Test_VirtualFileSystem_Lstat_InvalidPath(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fail()
		}
	}()
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	_, _ = fs.Lstat("/.")
}

func Test_VirtualFileSystem_Set_ReplacesFileContentsCorrectly(t *testing.T) {
	name := "/replacesfilecontents"
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		name: {Content: []byte("filecontentsorig")},
	})
	expected := []byte("filecontentsreplaces")
	fs.Set(name, &InMemoryFile{
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

func Test_VirtualFileSystem_Stat_ENOENT(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{})
	_, err := fs.Stat("/passwd")
	if err == nil || !os.IsNotExist(err) {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Stat_DirError2(t *testing.T) {
	errExpected := errors.New("unknown error 14")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
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

func Test_VirtualFileSystem_Stat_DirError3(t *testing.T) {
	errExpected := errors.New("unknown error 15")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
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

func Test_VirtualFileSystem_Stat_FileError(t *testing.T) {
	errExpected := errors.New("unknown error 12")
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/passwd": {Error: errExpected},
	})
	_, errActual := fs.Stat("/passwd")
	if errActual != errExpected {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Stat_ENOTDIR(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/enotdir2": {},
	})
	_, err := fs.Stat("/enotdir2/file3")
	if err != syscall.ENOTDIR {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Stat_TooManyLinks(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/selflink": {
			Content: []byte("selflink"),
			Mode:    os.ModeSymlink,
		},
	})
	_, err := fs.Stat("/selflink")
	if err != errTooManyLinks {
		t.Fail()
	}
}

func Test_VirtualFileSystem_Stat_AbsSymlink(t *testing.T) {
	fs := NewInMemoryUnixFileSystem(map[string]InMemoryFile{
		"/file": {},
		"/link": {
			Content: []byte("/file"),
			Mode:    os.ModeSymlink,
		},
	})
	_, err := fs.Stat("/link")
	if err != nil {
		t.Error(err)
	}
}

func Test_TrimTrailingSlashes_Success(t *testing.T) {
	ret := trimTrailingSlashes("/")
	if ret != "" {
		t.Fail()
	}
}
