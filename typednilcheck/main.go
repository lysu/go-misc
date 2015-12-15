package main

import (
	"os"
	"runtime"
	"github.com/lysu/go-misc/typednilcheck/internal/typednilcheck"
	"fmt"
	"flag"
	"github.com/kisielk/gotool"
	"strings"
)

const (
	exitCodeOk int = iota
	exitUncheckedError
	exitFatalError
)

var abspath bool

func MainCmd(args []string) int {
	runtime.GOMAXPROCS(runtime.NumCPU())

	checker := &typednilcheck.Checker{}
	paths, err := parseFlags(checker, args)
	if err != exitCodeOk {
		return err
	}

	if err := checker.CheckPackages(paths...); err != nil {
		if e, ok := err.(typednilcheck.PossibleTypedNilErrors); ok {
			reportPossibleNilErrors(e)
			return exitUncheckedError
		} else if err == typednilcheck.ErrNoGoFiles {
			fmt.Fprintln(os.Stderr, err)
			return exitCodeOk
		}
		fmt.Fprintf(os.Stderr, "error: failed to check packages: %s\n", err)
		return exitFatalError
	}
	return exitCodeOk
}

func reportPossibleNilErrors(e typednilcheck.PossibleTypedNilErrors) {
	for _, uncheckedError := range e.Errors {
		pos := uncheckedError.Pos.String()
		if !abspath {
			if i := strings.Index(pos, "/src/"); i != -1 {
				pos = pos[i+len("/src/"):]
			}
		}
		fmt.Printf("%s\t%s\t%s\n", pos, uncheckedError.Line, "| possible typed nil ---> " + uncheckedError.Symbol)
	}
}


func parseFlags(checker *typednilcheck.Checker, args []string) ([]string, int) {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	if err := flags.Parse(args[1:]); err != nil {
		return nil, exitFatalError
	}
	// ImportPaths normalizes paths and expands '...'
	return gotool.ImportPaths(flags.Args()), exitCodeOk
}

func main() {
	os.Exit(MainCmd(os.Args))
}
