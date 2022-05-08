// log is a package that summarizes the log output functions.
// If you want to output the log, rewrite the package variable.
package log

type Log func(args ...interface{})
type Logf func(formatMsg string, args ...interface{})

var (
	Debug  Log  = NilLog
	Debugf Logf = NilLogf
	Info   Log  = NilLog
	Infof  Logf = NilLogf
	Warn   Log  = NilLog
	Warnf  Logf = NilLogf
	Error  Log  = NilLog
	Errorf Logf = NilLogf
)

// NilLog is a function that outputs nothing.
func NilLog(args ...interface{}) {}

// NilLogf is a function that outputs nothing.
func NilLogf(formatMsg string, args ...interface{}) {}
