package main

import (
	"os"
	"time"

	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
)

func main() {

	r := reporter.New(os.Stdout)
	row1 := r.AddRow("row1")
	row2 := r.AddRow("row2")
	r.Refresh()
	time.Sleep(1 * time.Second)
	r.LogWriter().Write([]byte("log1\n"))
	r.Refresh()
	_ = row1
	_ = row2
}
