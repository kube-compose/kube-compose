package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kube-compose/kube-compose/internal/pkg/progress/reporter"
)

func main() {

	r := reporter.New(os.Stdout)
	for i := 1; i <= 10; i++ {
		r.AddRow(fmt.Sprintf("row%d", i))
		r.Refresh()
		time.Sleep(1 * time.Second)
		_, _ = r.LogWriter().Write([]byte(fmt.Sprintf("log%d\n", i)))
		r.Refresh()
		time.Sleep(1 * time.Second)
	}
}
