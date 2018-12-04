package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/ajankovic/xdiff"
	"github.com/ajankovic/xdiff/parser"
	"github.com/ajankovic/xdiff/xtree"
)

var (
	leftSource  string
	rightSource string
	cpuprofile  string
	memprofile  string
	version     string
	date        string
	showVersion bool
)

func main() {
	flag.BoolVar(&showVersion, "version", false, "show build information.")
	flag.StringVar(&leftSource, "left", "", "original source for comparison.")
	flag.StringVar(&rightSource, "right", "", "edited source for comparison.")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`.")
	flag.StringVar(&memprofile, "memprofile", "", "write memory profile to `file`.")
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		fmt.Println(date)
		os.Exit(0)
	}
	if leftSource == "" || rightSource == "" {
		fail("filepaths of both sources are required.")
	}
	li, err := os.Stat(leftSource)
	if err != nil {
		fail("can't access left source %s error: %v",
			leftSource, err.Error())
	}
	ri, err := os.Stat(rightSource)
	if err != nil {
		fail("can't access right source %s error: %v",
			rightSource, err.Error())
	}
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			fail("could not create CPU profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fail("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	var left, right *xtree.Node
	var wg sync.WaitGroup
	start := time.Now()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		p := parser.New()
		start := time.Now()

		if li.IsDir() {
			left, err = p.ParseDir(leftSource)
		} else {
			left, err = p.ParseFile(leftSource)
		}
		if err != nil {
			fail("failed to parse left file %s error: %v",
				leftSource, err.Error())
		}
		fmt.Fprintf(os.Stderr, "left parsing time: %s\n", time.Since(start))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		p := parser.New()
		start := time.Now()
		if ri.IsDir() {
			right, err = p.ParseDir(rightSource)
		} else {
			right, err = p.ParseFile(rightSource)
		}
		if err != nil {
			fail("failed to parse right file %s error: %v",
				rightSource, err.Error())
		}
		fmt.Fprintf(os.Stderr, "right parsing time: %s\n", time.Since(start))
	}()
	wg.Wait()
	fmt.Fprintf(os.Stderr, "total parsing time: %s\n", time.Since(start))
	start = time.Now()
	diff, err := xdiff.Compare(left, right)
	if err != nil {
		fail("failed to compare files error: %v", err.Error())
	}
	fmt.Fprintf(os.Stderr, "comparing time: %s\n", time.Since(start))
	enc := xdiff.NewTextEncoder(os.Stdout)
	if err := enc.Encode(diff); err != nil {
		fail("failed to generate output error: %v", err.Error())
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			fail("could not create memory profile: %v", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			fail("could not write memory profile: %v", err)
		}
		f.Close()
	}
}

func fail(msg string, params ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", params...)
	os.Exit(1)
}
