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

type AnimationType int

const (
	ansiiTerminalCommandEscape               = "\x1b"
	minProgressTaskColumnWidth               = 20
	animationSpeed                           = time.Second // length of the interval with which animation state is incremented
	AnimationTypeNone          AnimationType = 0
	AnimationTypeDockerPull    AnimationType = 1
	AnimationTypeDockerPush    AnimationType = 2
	RefreshInterval                          = 100 * time.Millisecond
)

var (
	progressBarChars = []string{
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
		AnimationType: AnimationTypeDockerPush,
		Text:          "pushing image",
		TextWidth:     13,
		Priority:      1,
	}
	StatusDockerPull = &Status{
		AnimationType: AnimationTypeDockerPull,
		Text:          "pulling image",
		TextWidth:     13,
		Priority:      1,
	}
	StatusWaiting = &Status{
		TextWidth: 7,
		Text:      "waiting",
		Priority:  0,
	}
	StatusRunning = &Status{
		Text:      "running 🐵⭐️", // monkey + star
		TextWidth: 12,
		Priority:  2,
	}
	StatusReady = &Status{
		Text:      "ready 🐵⭐️", // monkey + star
		TextWidth: 10,
		Priority:  3,
	}
)

type Status struct {
	AnimationType AnimationType
	Priority      int
	Text          string
	TextWidth     int
}

type Reporter struct {
	buffer                    *bytes.Buffer
	mutex                     sync.Mutex
	isTerminal                bool
	lastRefreshNumLines       int
	lastRefreshTime           time.Time
	logBuffer                 *bytes.Buffer
	logLinesSinceFirstRefresh int
	logWriter                 io.Writer
	rows                      []*ReporterRow
	animationState            int // 0, 1 or 2
	animationTime             time.Duration
	out                       *os.File
}

func New(out *os.File) *Reporter {
	r := &Reporter{
		buffer:     bytes.NewBuffer(make([]byte, 256)),
		isTerminal: out != nil && terminal.IsTerminal(int(out.Fd())),
		logBuffer:  bytes.NewBuffer(make([]byte, 256)),
		out:        out,
	}
	r.logWriter = &reporterLogWriter{
		r: r,
	}
	return r
}

func (r *Reporter) AddRow(name string) *ReporterRow {
	rr := &ReporterRow{
		name: name,
		r:    r,
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.rows = append(r.rows, rr)
	return rr
}

func (r *Reporter) IsTerminal() bool {
	return r.isTerminal
}

func (r *Reporter) LogWriter() io.Writer {
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

func (r *Reporter) updateTime() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefreshTime)
	r.animationTime += elapsed
	dividend := r.animationTime / animationSpeed
	r.animationTime -= dividend * animationSpeed
	r.animationState = (r.animationState + int(dividend)) % 3
	r.lastRefreshTime = now
}

func (r *Reporter) refresh() {
	r.updateTime()
	defer func() {
		if v := recover(); v != nil {
			writeError := v.(*writeError)
			if writeError != nil {
				fmt.Fprintln(os.Stderr, writeError.Error)
			} else {
				panic(v)
			}
		}
	}()
	r.buffer.Reset()
	_, terminalLines, err := terminal.GetSize(int(r.out.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting size of terminal: %v\n", err)
		return
	}
	offset := terminalLines - 1 - r.logLinesSinceFirstRefresh - r.lastRefreshNumLines
	if offset+r.refreshNumLines() <= 0 {
		r.flushLogs()
		return
	}
	if offset < 0 {
		r.writeCmd(fmt.Sprintf("[%dA", terminalLines-1)) // Move to first line of output...
	} else {
		r.writeCmd(fmt.Sprintf("[%dA", r.lastRefreshNumLines+r.logLinesSinceFirstRefresh)) // Move to first line of output...
	}
	columns := []column{
		column{
			name:  "service",
			width: len("service"),
		},
		column{
			name:  "status",
			width: len("status"),
		},
	}
	taskNameToColumnIndex := map[string]int{}
	for _, rr := range r.rows {
		width := len(rr.name)
		if width > columns[0].width {
			columns[0].width = width
		}
		status := rr.status()
		width = status.TextWidth
		if status.AnimationType != AnimationTypeNone {
			width += 9
		}
		if width > columns[1].width {
			columns[1].width = width
		}
		for _, pt := range rr.tasks {
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
			if vPercentInt == 100 {
				width += 3
			} else if vPercentInt >= 10 {
				width += 2
			} else {
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
				if vPercentInt == 100 {
					progressBarWidth -= 3
				} else if vPercentInt >= 10 {
					progressBarWidth -= 2
				} else {
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
		r.writeCmd("E") // Move next line
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
		r.writeCmd("E") // Move next line
	}
	offset++

	for _, rr := range r.rows {
		if offset >= 0 {
			r.writeCmd("[2K") // Clear entire line
			r.writef("%-*s", columns[0].width, rr.name)
			r.writef(" │ ")

			status := rr.status()
			width := columns[1].width
			r.writef("%s", status.Text)
			width -= status.TextWidth
			if status.AnimationType != AnimationTypeNone {
				r.writef(" ")
				r.writeAnimation(status.AnimationType)
				width -= 9
			}
			r.writeRepeated(" ", width)

			for i := 2; i < len(columns); i++ {
				j := 0
				for j < len(rr.tasks) {
					if rr.tasks[j].name == columns[i].name {
						break
					}
					j++
				}
				r.writef(" │ ")
				if j >= len(rr.tasks) {
					// No value for this cell
					r.writef("%*s", columns[i].width, "")
					continue
				}
				pt := rr.tasks[j]
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
			r.writeCmd("E") // Move next line
		}
		offset++
	}
	for i := r.refreshNumLines(); i < r.lastRefreshNumLines; i++ {
		if offset >= 0 {
			r.writeCmd("[2K") // Clear entire line
			r.writeCmd("E")   // Move next line
		}
		offset++
	}
	r.writeCmd(fmt.Sprintf("[%dB", r.logLinesSinceFirstRefresh))
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
	r.logLinesSinceFirstRefresh += bytes.Count(r.logBuffer.Bytes(), []byte{'\n'})
	_, err := io.Copy(r.out, r.logBuffer)
	handleError(err)
	r.logBuffer.Reset()
}

func (r *Reporter) writeAnimation(at AnimationType) {
	switch at {
	case AnimationTypeDockerPush:
		r.writef("👋") // Wave
		r.writeRepeated("  ", r.animationState)
		r.writef("🐳") // Whale
		r.writeRepeated("  ", 2-r.animationState)
	case AnimationTypeDockerPull:
		r.writef("🧲") // Magnet
		r.writeRepeated("  ", 2-r.animationState)
		r.writef("🐳") // Whale
		r.writeRepeated("  ", r.animationState)
	}
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
	_, err := fmt.Fprintf(r.buffer, "%s%s", ansiiTerminalCommandEscape, s)
	handleError(err)
}

type ReporterRow struct {
	name     string
	r        *Reporter
	tasks    []*ProgressTask
	statuses []*Status
}

func (rr *ReporterRow) AddProgressTask(name string) *ProgressTask {
	pt := &ProgressTask{
		name: name,
		rr:   rr,
	}
	rr.r.mutex.Lock()
	defer rr.r.mutex.Unlock()
	rr.tasks = append(rr.tasks, pt)
	return pt
}

func (rr *ReporterRow) AddStatus(s *Status) {
	rr.r.mutex.Lock()
	defer rr.r.mutex.Unlock()
	i := rr.statusBinarySearch(s.Priority)
	if i < 0 {
		i = -i - 1
	}
	rr.statuses = append(rr.statuses[:i], append([]*Status{s}, rr.statuses[i:]...)...)
}

func (rr *ReporterRow) Name() string {
	return rr.name
}

func (rr *ReporterRow) RemoveStatus(s *Status) {
	rr.r.mutex.Lock()
	defer rr.r.mutex.Unlock()
	i := rr.statusBinarySearch(s.Priority)
	if i < 0 {
		return
	}
	iLast := len(rr.statuses) - 1
	for {
		if rr.statuses[i] == s {
			copy(rr.statuses[i:], rr.statuses[i+1:])
			rr.statuses[iLast] = nil
			rr.statuses = rr.statuses[:iLast]
			return
		}
		i++
		if i > iLast || rr.statuses[i].Priority != s.Priority {
			break
		}
	}
}

func (rr *ReporterRow) status() *Status {
	if len(rr.statuses) > 0 {
		return rr.statuses[len(rr.statuses)-1]
	}
	return StatusWaiting
}

func (rr *ReporterRow) statusBinarySearch(priority int) int {
	lo := 0
	hi := len(rr.statuses) - 1
	for lo <= hi {
		mi := lo + (hi-lo)/2
		if rr.statuses[mi].Priority > priority {
			hi = mi - 1
		} else if rr.statuses[mi].Priority < priority {
			lo = mi + 1
		} else {
			return lo
		}
	}
	return -lo - 1
}

type ProgressTask struct {
	done bool
	name string
	rr   *ReporterRow
	v    float64
}

func (pt *ProgressTask) Done() {
	if pt.done {
		return
	}
	pt.rr.r.mutex.Lock()
	defer pt.rr.r.mutex.Unlock()
	pt.done = true
	tasks := pt.rr.tasks
	iLast := len(tasks) - 1
	for i := 0; i <= iLast; i++ {
		if tasks[i] == pt {
			tasks[i], tasks[iLast] = tasks[iLast], tasks[i]
			pt.rr.tasks = tasks[0:iLast]
			break
		}
	}
}

func (pt *ProgressTask) Name() string {
	return pt.name
}

func (pt *ProgressTask) Update(v float64) {
	if !pt.rr.r.isTerminal {
		return
	}
	if v < 0 {
		v = 0
	} else if v > 1 {
		v = 1
	}
	pt.rr.r.mutex.Lock()
	defer pt.rr.r.mutex.Unlock()
	pt.v = v
}

type reporterLogWriter struct {
	r *Reporter
}

func (rlw *reporterLogWriter) Write(b []byte) (n int, err error) {
	return rlw.r.writeLogs(b)
}
