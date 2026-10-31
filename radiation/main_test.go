package radiation_test

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"
)

// failed is the status a run that failed stops with. It is defined by the
// shell.
const failed = 1

// noTimeout is the -test.timeout of a run that is never stopped. It is
// defined by go test: -timeout 0 turns the timeout off.
const noTimeout = 0

// timeoutQuarter divides the timeout into the part taken off it. It is defined by
// the arithmetic.
const timeoutQuarter = 4

// everyStack is the debug level of runtime/pprof that writes a stack for each
// goroutine, the running test among them. It is defined by that package.
const everyStack = 2

// timeoutOf is the -test.timeout this binary was started with. go test
// gives every test binary one, ten minutes unless told otherwise.
func timeoutOf() time.Duration {
	got, _ := flag.Lookup("test.timeout").Value.(flag.Getter).Get().(time.Duration)
	return got
}

// overrun ends the run as failed with a message of its own. It comes before
// the panic go test raises at its timeout. Loops that do not end get here.
func overrun() {
	fmt.Fprintln(os.Stderr, "FAIL: the tests ran past three quarters of "+
		"their -test.timeout; a loop does not end")
	fmt.Fprintln(os.Stderr, "the goroutines, the running test among them:")
	pprof.Lookup("goroutine").WriteTo(os.Stderr, everyStack)
	os.Exit(failed)
}

// TestMain runs the tests. Unless there is no -test.timeout it arms overrun
// at three quarters of it. The mutation run gives twenty seconds, so a
// mutant with an endless loop is killed at fifteen and does not time out.
// Ordinary runs have the ten minutes of go test. Debuggers can use
// -timeout 0.
func TestMain(suite *testing.M) {
	flag.Parse()
	if timeout := timeoutOf(); timeout > noTimeout {
		time.AfterFunc(timeout-timeout/timeoutQuarter, overrun)
	}
	os.Exit(suite.Run())
}
