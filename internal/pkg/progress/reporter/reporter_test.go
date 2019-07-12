package reporter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func Test_Reporter_AddRow_Success(t *testing.T) {
	r := New(os.Stdout)
	name := "reporteraddrowsuccess"
	row := r.AddRow(name)
	if !reflect.DeepEqual(r.rows, []*Row{row}) {
		t.Fail()
	}
}

func Test_Reporter_IsTerminal(t *testing.T) {
	r := New(os.Stdout)
	r.IsTerminal()
}

func Test_IsTerminal_ReporterLogWriter(t *testing.T) {
	r := New(os.Stdout)
	rlw := &reporterLogWriter{
		r: r,
	}
	IsTerminal(rlw)
}

func Test_GetTerminalSize(t *testing.T) {
	r := New(os.Stdout)
	rlw := &reporterLogWriter{
		r: r,
	}
	_, _, _ = getTerminalSizeFunction(rlw)
}

func Test_Reporter_Refresh_NotTerminalSuccess(t *testing.T) {
	r := New(bytes.NewBuffer([]byte{}))
	r.Refresh()
}

func Test_Reporter_Refresh_GetTerminalSizeError(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		term.getSizeError = fmt.Errorf("refreshgetterminalsizeerror")
		r := New(term)
		r.Refresh()
	})
}
func Test_Reporter_Refresh_FlushesLogsIfTerminalHeightIsZero(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		term.height = 0
		expected := "FlushesLogsIfTerminalHeightIsZero"
		r := New(term)
		_, _ = r.LogSink().Write([]byte(expected))
		r.Refresh()
		actual := term.String()
		if actual != expected+"\n" {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_FlushesLogsIfNoRowsAreRendered(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		term.height = 1
		expected := "FlushesLogsIfNoRowsAreRendered"
		r := New(term)
		r.logLines = 2
		_, _ = r.LogSink().Write([]byte(expected))
		r.Refresh()
		actual := term.String()
		if actual != expected+"\n" {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_EmptyTableSuccess(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		term.height = 1
		r := New(term)
		r.Refresh()
		actual := term.String()
		if actual != "service │ status\n────────┼───────\n" {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_GrowSuccess(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		r := New(term)
		r1 := r.AddRow("row1")
		r1.AddStatus(StatusDockerPull)
		r1t1 := r1.AddProgressTask("pulling image")
		r1t1.Update(0.05)
		_, _ = r.LogErrorSink().Write([]byte("log1"))
		r.Refresh()

		r2 := r.AddRow("longrowname")
		r2.AddStatus(StatusDockerPush)
		r2t1 := r2.AddProgressTask("pushing image")
		r2t1.Update(0.5)
		_, _ = r.LogErrorSink().Write([]byte("log2"))
		r1t1.Update(1)
		r.Refresh()
		actual := term.String()
		expected := "service     │ status        │ pulling image        │ pushing image       \n" +
			"────────────┼───────────────┼──────────────────────┼─────────────────────\n" +
			"row1        │ pulling image │ ███████████████ 100% │                     \n" +
			"longrowname │ pushing image │                      │ ████████         50%\n" +
			"log2\n"
		if actual != expected {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_ShrinkSuccess(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		r := New(term)
		r1 := r.AddRow("row1")
		r1.AddStatus(StatusDockerPull)
		r1t1 := r1.AddProgressTask("pulling image")
		r1t1.Update(0.05)
		r2 := r.AddRow("row_2")
		r2.AddStatus(StatusDockerPush)
		r2t1 := r2.AddProgressTask("x")
		r2t1.Update(0.5)
		_, _ = r.LogErrorSink().Write([]byte("log1"))
		r.Refresh()
		r.DeleteRow(r2)
		r.Refresh()
		actual := term.String()
		expected := "service │ status        │ pulling image       \n" +
			"────────┼───────────────┼─────────────────────\n" +
			"row1    │ pulling image │ ▉                 5%\n" +
			"\n" +
			"log1\n"
		if actual != expected {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_NegativeOffsetSuccess(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		term.height = 3
		r := New(term)
		_, _ = r.LogErrorSink().Write([]byte("log1\n"))
		r.Refresh()
		r.Refresh()
		actual := term.String()
		expected := "service │ status\n" +
			"────────┼───────\n" +
			"log1\n"
		if actual != expected {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
}

func Test_Reporter_Refresh_WriteError(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		r := New(term)
		term.writeError = fmt.Errorf("reporterrefreshwriteerror")
		_, _ = r.LogSink().Write([]byte("reporterrefreshwriteerrorlog"))
		r.Refresh()
	})
}

func Test_Reporter_Refresh_Panic(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		expected := fmt.Errorf("reporterrefreshpanic")
		defer func() {
			if v := recover(); v != expected {
				t.Fail()
			}
		}()
		r := New(term)
		term.panic = expected
		_, _ = r.LogSink().Write([]byte("reporterrefreshpaniclog"))
		r.Refresh()
	})
}

func Test_Row_Name(t *testing.T) {
	expected := "rowname"
	row := &Row{
		name: expected,
	}
	if row.Name() != expected {
		t.Fail()
	}
}

func Test_Row_StatusBinarySearch_SuccessNotFound(t *testing.T) {
	row := &Row{
		statuses: []*Status{},
	}
	i := row.statusBinarySearch(0)
	if i != -1 {
		t.Fail()
	}
}

func Test_Row_RemoveStatus_NotFound1(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("asdf")
	if row.RemoveStatus(StatusWaiting) {
		t.Fail()
	}
}
func Test_Row_RemoveStatus_NotFound2(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("asdf")
	row.AddStatus(&Status{
		Priority: StatusWaiting.Priority,
	})
	if row.RemoveStatus(StatusWaiting) {
		t.Fail()
	}
}

func Test_Row_RemoveStatus_Success(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("asdf")
	row.AddStatus(StatusWaiting)
	if !row.RemoveStatus(StatusWaiting) {
		t.Fail()
	}
}

func Test_Row_RemoveStatus_NotFound(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("asdf")
	row.RemoveStatus(StatusWaiting)
}

func Test_Row_Status_Default(t *testing.T) {
	row := &Row{}
	if row.status() != StatusWaiting {
		t.Fail()
	}
}

func Test_ProgressTask_Done(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("progresstaskdonerow")
	pt1 := row.AddProgressTask("progresstaskdonept")
	row.AddProgressTask("progresstaskdonept")
	pt1.Done()
	pt1.Done()
	if len(row.tasks) != 1 {
		t.Fail()
	}
}

func Test_ProgressTask_Name(t *testing.T) {
	expected := "progresstaskname"
	pt := &ProgressTask{
		name: expected,
	}
	if pt.Name() != expected {
		t.Fail()
	}
}

func Test_ProgressTask_Update_Min(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("progresstaskupdatemin")
	pt := &ProgressTask{
		v:   1.0,
		row: row,
	}
	pt.Update(-1.0)
	if pt.v != 0 {
		t.Fail()
	}
}

func Test_ProgressTask_Update_Max(t *testing.T) {
	r := New(os.Stdout)
	row := r.AddRow("progresstaskupdatemax")
	pt := &ProgressTask{
		row: row,
	}
	pt.Update(2.0)
	if pt.v != 1.0 {
		t.Fail()
	}
}

func Test_Row_StatusBinarySearch_SuccessFound(t *testing.T) {
	r := &Row{
		statuses: []*Status{
			{
				Priority: 0,
			},
			{
				Priority: 1,
			},
			{
				Priority: 2,
			},
			{
				Priority: 3,
			},
			{
				Priority: 4,
			},
		},
	}
	i := r.statusBinarySearch(1)
	if i != 1 {
		t.Fail()
	}
}

func withMockTerminal(cb func(term *mockTerminal)) {
	term := newMockTerminal()
	term.height = 20
	defer func() {
		isTerminalFunction = IsTerminal
		getTerminalSizeFunction = GetTerminalSize
	}()
	isTerminalFunction = mockIsTerminal
	getTerminalSizeFunction = mockGetTerminalSize
	cb(term)
}

type mockTerminal struct {
	height       int
	getSizeError error
	writeError   error
	panic        interface{}
	line         int
	column       int
	state        int
	// Buffer for decimal numbers that are being parsed...
	dec int
	// Number of digits in decimal number.,
	decLen int
	data   [][]byte
}

func newMockTerminal() *mockTerminal {
	return &mockTerminal{
		data: [][]byte{},
	}
}

func (term *mockTerminal) String() string {
	var sb strings.Builder
	for _, lineData := range term.data {
		sb.Write(lineData)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (term *mockTerminal) clearLine() {
	if term.line < len(term.data) {
		term.data[term.line] = term.data[term.line][:0]
	}
}

func (term *mockTerminal) nextLine() {
	term.line++
	term.column = 0
}

func (term *mockTerminal) writeRawChar(b byte) {
	var lineData []byte
	if term.line >= len(term.data) {
		c := cap(term.data)
		if term.line >= c {
			c *= 2
			if c == 0 {
				c = 4
			}
			data := make([][]byte, c)
			copy(data, term.data)
			term.data = data
		}
		term.data = term.data[:term.line+1]
		lineData = term.data[term.line][:0]
		term.data[term.line] = lineData
	} else {
		lineData = term.data[term.line]
	}
	if term.column >= len(lineData) {
		c := cap(lineData)
		if term.column >= c {
			c *= 2
			if c == 0 {
				c = 64
			}
			lineDataGrown := make([]byte, c)
			copy(lineDataGrown, lineData)
			lineData = lineDataGrown[:term.column+1]
		} else {
			lineData = lineData[:term.column+1]
		}
		term.data[term.line] = lineData
	}
	lineData[term.column] = b
	term.column++
}

// TODO https://github.com/kube-compose/kube-compose/issues/227 reduce cyclomatic complexity of this function
//nolint
func (term *mockTerminal) Write(p []byte) (int, error) {
	if term.writeError != nil {
		return 0, term.writeError
	}
	if term.panic != nil {
		panic(term.panic)
	}
	for i := 0; i < len(p); i++ {
		switch term.state {
		case 0:
			switch p[i] {
			case ansiiTerminalCommandEscape:
				term.state = 1
			case '\n':
				term.nextLine()
			case '\r':
				term.column = 0
			default:
				term.writeRawChar(p[i])
			}
		case 1:
			switch p[i] {
			case 'E':
				term.nextLine()
				term.state = 0
			case '[':
				term.state = 2
			default:
				panic(fmt.Errorf("unexpected escaped byte (state1) %s (0x%x)", string(p[i]), p[i]))
			}
		case 2:
			if p[i] >= '0' && p[i] <= '9' {
				term.dec = int(p[i] - '0')
				term.decLen = 1
				term.state = 3
			} else {
				panic(fmt.Errorf("unexpected escaped byte (state2) %s (0x%x)", string(p[i]), p[i]))
			}
		case 3:
			switch {
			case p[i] >= '0' && p[i] <= '9':
				if term.dec > 214748363 {
					panic(fmt.Errorf("integer overflow"))
				}
				term.dec = term.dec*10 + int(p[i]-'0')
				term.decLen++
			case p[i] == 'K' && term.dec == 2 && term.decLen == 1:
				term.clearLine()
				term.state = 0
			case p[i] == 'A':
				if term.dec > term.line {
					panic(fmt.Errorf("attempted to move above first line, should this be interpreted as move to line 0?"))
				}
				term.line -= term.dec
				term.state = 0
			case p[i] == 'B':
				term.line += term.dec
				term.state = 0
			default:
				panic(fmt.Errorf("unexpected escaped byte (state3) %s (0x%x) (dec=%d, decLen=%d)", string(p[i]), p[i], term.dec, term.decLen))
			}
		}
	}
	return len(p), nil
}

func mockIsTerminal(w io.Writer) bool {
	_, ok := w.(*mockTerminal)
	return ok
}

func mockGetTerminalSize(w io.Writer) (width, height int, err error) {
	term := w.(*mockTerminal)
	if term.getSizeError != nil {
		err = term.getSizeError
		return
	}
	height = term.height
	return
}
