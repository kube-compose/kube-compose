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
	defaultStatus = &Status{
		Width:    12,
		Text:     "🙈💣 unknown", // monkey see no evil + bomb
		Priority: 0,
	}
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
)

type Reporter struct {
	buffer              *bytes.Buffer
	mutex               sync.Mutex
	isTerminal          bool
	lastRefreshNumLines int
	lastRefreshTime     time.Time
	rows                []*ReporterRow
	taskAnimationTypes  map[string]AnimationType
	animationState      int // 0, 1 or 2
	animationTime       time.Duration
	out                 *os.File
}

func New(out *os.File) *Reporter {
	return &Reporter{
		buffer:             bytes.NewBuffer(make([]byte, 256)),
		isTerminal:         out != nil && terminal.IsTerminal(int(out.Fd())),
		out:                out,
		taskAnimationTypes: map[string]AnimationType{},
	}
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

func (r *Reporter) GetTaskAnimationType(taskName string) AnimationType {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.taskAnimationTypes[taskName]
}

func (r *Reporter) Refresh() {
	if !r.isTerminal {
		return
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.refresh()
}

func (r *Reporter) SetTaskAnimationType(taskName string, at AnimationType) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if at == AnimationTypeNone {
		delete(r.taskAnimationTypes, taskName)
	} else {
		r.taskAnimationTypes[taskName] = at
	}
}

type column struct {
	headerDesiredWidth int
	at                 AnimationType
	name               string
	progressBarWidth   int
	width              int
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

type Status struct {
	Width    int
	Text     string
	Priority int
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
	for i := 0; i < r.lastRefreshNumLines; i++ {
		r.writeCmd("[1A") // Move up one line
		r.writeCmd("[2K") // Clear entire line
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
		if status.Width > columns[1].width {
			columns[1].width = status.Width
		}
		for _, pt := range rr.tasks {
			columnIndex, ok := taskNameToColumnIndex[pt.name]
			if !ok {
				columnIndex = len(columns)
				at := r.taskAnimationTypes[pt.name]
				headerDesiredWidth := len(pt.name)
				if at != AnimationTypeNone {
					headerDesiredWidth += 9
				}
				columns = append(columns, column{
					headerDesiredWidth: headerDesiredWidth,
					at:                 at,
					name:               pt.name,
					width:              headerDesiredWidth,
					progressBarWidth:   int(^uint(0) >> 1),
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
			if columns[columnIndex].headerDesiredWidth > width {
				// Scale progress bar width with name of task
				progressBarWidth = columns[columnIndex].headerDesiredWidth - width
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
	r.writef("%-*s", columns[0].width, columns[0].name)
	for i := 1; i < len(columns); i++ {
		column := &columns[i]
		width := column.width
		r.writef(" │ ")

		width -= len(column.name)
		r.writef(column.name)
		if column.at != AnimationTypeNone {
			r.writef(" ")
			r.writeAnimation(column.at)
			width -= 9
		}
		r.writeRepeated(" ", width)
	}
	r.writef("\n")
	// separator row
	r.writeRepeated("─", columns[0].width)
	for i := 1; i < len(columns); i++ {
		r.writef("─┼─")
		r.writeRepeated("─", columns[i].width)
	}
	r.writef("\n")
	r.lastRefreshNumLines = len(r.rows) + 2
	for _, rr := range r.rows {
		r.writef("%-*s", columns[0].width, rr.name)
		r.writef(" │ ")
		status := rr.status()
		r.writef("%s", status.Text)
		r.writeRepeated(" ", columns[1].width-status.Width)
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
		r.writef("\n")
	}

	r.flush()
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
	return defaultStatus
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
