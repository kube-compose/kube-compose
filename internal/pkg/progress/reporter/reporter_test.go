package reporter

import (
	"os"
	"reflect"
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

func Test_Reporter_Refresh(t *testing.T) {
	r := New(os.Stdout)
	r.Refresh()
}
