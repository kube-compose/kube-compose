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
	getTerminalSizeFunction(rlw)
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

func Test_Reporter_Refresh_RowsSuccess(t *testing.T) {
	withMockTerminal(func(term *mockTerminal) {
		r := New(term)
		r1 := r.AddRow("row1")
		r1.AddStatus(StatusDockerPull)
		r1t1 := r1.AddProgressTask("pulling image")
		r1t1.Update(0.5)
		_, _ = r.LogErrorSink().Write([]byte("log1"))
		r.Refresh()

		r2 := r.AddRow("row2")
		r2.AddStatus(StatusDockerPush)
		r2t1 := r2.AddProgressTask("pushing image")
		r2t1.Update(0.5)
		_, _ = r.LogErrorSink().Write([]byte("log2"))
		r.Refresh()
		actual := term.String()
		expected := "service │ status        │ pulling image        │ pushing image       \n" +
			"────────┼───────────────┼──────────────────────┼─────────────────────\n" +
			"row1    │ pulling image │ ████████         50% │                     \n" +
			"row2    │ pushing image │                      │ ████████         50%\n" +
			"log2\n"
		if actual != expected {
			t.Log(actual)
			t.Log("end")
			t.Logf("%#v", actual)
			t.Fail()
		}
	})
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

func (term *mockTerminal) Write(p []byte) (int, error) {
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

func mockGetTerminalSize(w io.Writer) (width int, height int, err error) {
	term := w.(*mockTerminal)
	if term.getSizeError != nil {
		err = term.getSizeError
		return
	}
	height = term.height
	return
}
