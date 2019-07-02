package reporter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	ansiiTerminalCommandEscape = '\x1b'
	minProgressTaskColumnWidth = 20
	RefreshInterval            = 100 * time.Millisecond
)

var (
	isTerminalFunction      = IsTerminal
	getTerminalSizeFunction = GetTerminalSize
	progressBarChars        = []string{
		" ",
		"▏",
		"▎",
		"▍",
		"▌",
		"▋",
		"▊",
		"▉",
		"█",
	}
	StatusDockerPush = &Status{
		Text:      "pushing image",
		TextWidth: 13,
		Priority:  1,
	}
	StatusDockerPull = &Status{
		Text:      "pulling image",
		TextWidth: 13,
		Priority:  1,
	}
	StatusWaiting = &Status{
		TextWidth: 7,
		Text:      "waiting",
		Priority:  0,
	}
	StatusRunning = &Status{
		Text:      "running ⭐️", // star
		TextWidth: 10,
		Priority:  2,
	}
	StatusReady = &Status{
		Text:      "ready ⭐️", // star
		TextWidth: 8,
		Priority:  3,
	}
)

type Status struct {
	Priority  int
	Text      string
	TextWidth int
}

type Reporter struct {
	buffer              *bytes.Buffer
	mutex               sync.Mutex
	isTerminal          bool
	lastRefreshNumLines int
	logBuffer           *bytes.Buffer
	logLines            int
	logWriter           io.Writer
	rows                []*Row
	out                 io.Writer
}

func New(out io.Writer) *Reporter {
	r := &Reporter{
		buffer:     bytes.NewBuffer([]byte{}),
		isTerminal: isTerminalFunction(out),
		logBuffer:  bytes.NewBuffer([]byte{}),
		out:        out,
	}
	r.logWriter = &reporterLogWriter{
		r: r,
	}
	return r
}

func (r *Reporter) AddRow(name string) *Row {
	row := &Row{
		name: name,
		r:    r,
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.rows = append(r.rows, row)
	return row
}

func GetTerminalSize(w io.Writer) (width int, height int, err error) {
	if rlw, ok := w.(*reporterLogWriter); ok {
		return GetTerminalSize(rlw.r.out)
	}
	return terminal.GetSize(int(w.(*os.File).Fd()))
}

func IsTerminal(w io.Writer) bool {
	if rlw, ok := w.(*reporterLogWriter); ok {
		return IsTerminal(rlw.r.out)
	}
	if file, ok := w.(*os.File); ok {
		return terminal.IsTerminal(int(file.Fd()))
	}
	return false
}

func (r *Reporter) IsTerminal() bool {
	return r.isTerminal
}

func (r *Reporter) LogSink() io.Writer {
	return r.logWriter
}

func (r *Reporter) LogErrorSink() io.Writer {
	return r.logWriter
}

func (r *Reporter) Refresh() {
	if !r.isTerminal {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refresh()
}

type column struct {
	name             string
	progressBarWidth int
	width            int
}

func (r *Reporter) refresh() {
	defer func() {
		if v := recover(); v != nil {
			if writeError, ok := v.(*writeError); ok {
				fmt.Fprintln(os.Stderr, writeError.Error)
			} else {
				panic(v)
			}
		}
	}()
	_, terminalLines, err := getTerminalSizeFunction(r.out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting size of terminal: %v\n", err)
		return
	}
	if terminalLines == 0 {
		r.flushLogs()
		return
	}
	offset := terminalLines - 1 - r.logLines - r.lastRefreshNumLines
	if offset+r.refreshNumLines() <= 0 {
		r.flushLogs()
		return
	}
	r.buffer.Reset()
	switch {
	case r.lastRefreshNumLines == 0:
		// This is required to start on the correct line
		r.writef("")
	case offset < 0:
		// Move to first line of output
		r.writeCmd(fmt.Sprintf("[%dA", terminalLines-1))
	default:
		// Move to first line of output
		r.writeCmd(fmt.Sprintf("[%dA", r.lastRefreshNumLines+r.logLines))
	}
	r.writef("\r")
	columns := []column{
		{
			name:  "service",
			width: len("service"),
		},
		{
			name:  "status",
			width: len("status"),
		},
	}
	taskNameToColumnIndex := map[string]int{}
	for _, row := range r.rows {
		width := len(row.name)
		if width > columns[0].width {
			columns[0].width = width
		}
		status := row.status()
		width = status.TextWidth
		if width > columns[1].width {
			columns[1].width = width
		}
		for _, pt := range row.tasks {
			columnIndex, ok := taskNameToColumnIndex[pt.name]
			if !ok {
				columnIndex = len(columns)
				columns = append(columns, column{
					name:             pt.name,
					width:            len(pt.name),
					progressBarWidth: int(^uint(0) >> 1),
				})
				taskNameToColumnIndex[pt.name] = columnIndex
			}
			// Calculate width of content
			width := 2 // len(" " + "%") == 2
			vPercentInt := int(pt.v * 100)
			// Add number of digits of percent to width needed
			switch {
			case vPercentInt == 100:
				width += 3
			case vPercentInt >= 10:
				width += 2
			default:
				width++
			}
			var progressBarWidth int
			if len(pt.name) > width {
				// Scale progress bar width with name of task
				progressBarWidth = len(pt.name) - width
			} else {
				// But progress bar is at least 1
				progressBarWidth = 1
			}
			width += progressBarWidth
			if width < minProgressTaskColumnWidth {
				width = minProgressTaskColumnWidth
				progressBarWidth = width - 2
				switch {
				case vPercentInt == 100:
					progressBarWidth -= 3
				case vPercentInt >= 10:
					progressBarWidth -= 2
				default:
					progressBarWidth--
				}
			}
			if progressBarWidth < columns[columnIndex].progressBarWidth {
				columns[columnIndex].progressBarWidth = progressBarWidth
			}
			if columns[columnIndex].width < width {
				columns[columnIndex].width = width
			}
		}
	}
	// header row
	if offset >= 0 {
		r.writeCmd("[2K") // Clear entire line
		r.writef("%-*s", columns[0].width, columns[0].name)
		for i := 1; i < len(columns); i++ {
			column := &columns[i]
			width := column.width
			r.writef(" │ ")
			width -= len(column.name)
			r.writef(column.name)
			r.writeRepeated(" ", width)
		}
		r.writeCmd("E") // Next line
	}
	offset++

	// separator row
	if offset >= 0 {
		r.writeCmd("[2K") // Clear entire line
		r.writeRepeated("─", columns[0].width)
		for i := 1; i < len(columns); i++ {
			r.writef("─┼─")
			r.writeRepeated("─", columns[i].width)
		}
		r.writeCmd("E") // Next line
	}
	offset++

	for _, row := range r.rows {
		if offset >= 0 {
			r.writeCmd("[2K") // Clear entire line
			r.writef("%-*s", columns[0].width, row.name)
			r.writef(" │ ")

			status := row.status()
			width := columns[1].width
			r.writef("%s", status.Text)
			width -= status.TextWidth
			r.writeRepeated(" ", width)

			for i := 2; i < len(columns); i++ {
				j := 0
				for j < len(row.tasks) {
					if row.tasks[j].name == columns[i].name {
						break
					}
					j++
				}
				r.writef(" │ ")
				if j >= len(row.tasks) {
					// No value for this cell
					r.writef("%*s", columns[i].width, "")
					continue
				}
				pt := row.tasks[j]
				width := columns[i].width
				progressBarWidth := columns[i].progressBarWidth

				width -= progressBarWidth
				n := float64(progressBarWidth) * pt.v
				nInt := int(n)
				r.writeRepeated(progressBarChars[len(progressBarChars)-1], nInt)
				if pt.v != 1 {
					nFrac := n - float64(nInt)
					i := int(nFrac * float64(len(progressBarChars)))
					r.writef(progressBarChars[i])
					r.writeRepeated(" ", progressBarWidth-nInt-1)
				}
				vPercentInt := int(pt.v * 100)
				s := fmt.Sprintf("%d%%", vPercentInt)
				r.writeRepeated(" ", width-len(s))
				r.writef("%s", s)
			}
			r.writeCmd("E") // Next line
		}
		offset++
	}
	for i := r.refreshNumLines(); i < r.lastRefreshNumLines; i++ {
		if offset >= 0 {
			r.writeCmd("[2K") // Clear entire line
			r.writeCmd("[1B") // Move down one line
		}
		offset++
	}
	r.logLines -= r.refreshNumLines() - r.lastRefreshNumLines
	if r.logLines < 0 {
		r.logLines = 0
	} else if r.logLines > 0 {
		r.writeCmd(fmt.Sprintf("[%dB", r.logLines))
	}
	r.lastRefreshNumLines = r.refreshNumLines()
	r.flush()

	r.flushLogs()
}

func (r *Reporter) refreshNumLines() int {
	return len(r.rows) + 2 // the number of rows to be rendered in the reporter table
}

type writeError struct {
	Error error
}

func handleError(err error) {
	if err == nil {
		return
	}
	panic(&writeError{
		Error: err,
	})
}

func (r *Reporter) flush() {
	_, err := io.Copy(r.out, r.buffer)
	handleError(err)
}

func (r *Reporter) flushLogs() {
	r.logLines += bytes.Count(r.logBuffer.Bytes(), []byte{'\n'})
	_, err := io.Copy(r.out, r.logBuffer)
	handleError(err)
	r.logBuffer.Reset()
}

func (r *Reporter) writeRepeated(s string, n int) {
	for i := 0; i < n; i++ {
		r.writef(s)
	}
}

func (r *Reporter) writeLogs(b []byte) (n int, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	n, err = r.logBuffer.Write(b)
	return
}

func (r *Reporter) writef(format string, args ...interface{}) {
	_, err := fmt.Fprintf(r.buffer, format, args...)
	handleError(err)
}

func (r *Reporter) writeCmd(s string) {
	_, err := fmt.Fprintf(r.buffer, "%s%s", string(ansiiTerminalCommandEscape), s)
	handleError(err)
}

type Row struct {
	name     string
	r        *Reporter
	tasks    []*ProgressTask
	statuses []*Status
}

func (row *Row) AddProgressTask(name string) *ProgressTask {
	pt := &ProgressTask{
		name: name,
		row:  row,
	}
	row.r.mutex.Lock()
	defer row.r.mutex.Unlock()
	row.tasks = append(row.tasks, pt)
	return pt
}

func (row *Row) AddStatus(s *Status) {
	row.r.mutex.Lock()
	defer row.r.mutex.Unlock()
	i := row.statusBinarySearch(s.Priority)
	if i < 0 {
		i = -i - 1
	}
	row.statuses = append(row.statuses[:i], append([]*Status{s}, row.statuses[i:]...)...)
}

func (row *Row) Name() string {
	return row.name
}

func (row *Row) RemoveStatus(s *Status) {
	row.r.mutex.Lock()
	defer row.r.mutex.Unlock()
	i := row.statusBinarySearch(s.Priority)
	if i < 0 {
		return
	}
	iLast := len(row.statuses) - 1
	for {
		if row.statuses[i] == s {
			copy(row.statuses[i:], row.statuses[i+1:])
			row.statuses[iLast] = nil
			row.statuses = row.statuses[:iLast]
			return
		}
		i++
		if i > iLast || row.statuses[i].Priority != s.Priority {
			break
		}
	}
}

func (row *Row) status() *Status {
	if len(row.statuses) > 0 {
		return row.statuses[len(row.statuses)-1]
	}
	return StatusWaiting
}

func (row *Row) statusBinarySearch(priority int) int {
	lo := 0
	hi := len(row.statuses) - 1
	for lo <= hi {
		mi := lo + (hi-lo)/2
		switch {
		case row.statuses[mi].Priority > priority:
			hi = mi - 1
		case row.statuses[mi].Priority < priority:
			hi = mi + 1
		default:
			return lo
		}
	}
	return -lo - 1
}

type ProgressTask struct {
	done bool
	name string
	row  *Row
	v    float64
}

func (pt *ProgressTask) Done() {
	if pt.done {
		return
	}
	pt.row.r.mutex.Lock()
	defer pt.row.r.mutex.Unlock()
	pt.done = true
	tasks := pt.row.tasks
	iLast := len(tasks) - 1
	for i := 0; i <= iLast; i++ {
		if tasks[i] == pt {
			tasks[i], tasks[iLast] = tasks[iLast], tasks[i]
			pt.row.tasks = tasks[0:iLast]
			break
		}
	}
}

func (pt *ProgressTask) Name() string {
	return pt.name
}

func (pt *ProgressTask) Update(v float64) {
	if !pt.row.r.isTerminal {
		return
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	pt.row.r.mutex.Lock()
	defer pt.row.r.mutex.Unlock()
	pt.v = v
}

type reporterLogWriter struct {
	r *Reporter
}

func (rlw *reporterLogWriter) Write(b []byte) (n int, err error) {
	return rlw.r.writeLogs(b)
}
