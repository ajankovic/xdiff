# X-Diff algorithm implementation in Go

[![GoDoc Badge]][GoDoc] [![GoReportCard Badge]][GoReportCard]

*WORK IN PROGRESS*

Implementation of the algorithm described by [this paper](http://pages.cs.wisc.edu/~yuanwang/papers/xdiff.pdf)

You can use this package as both library and command line tool for diffing xml files. Project is still under havy development so expect braking changes until it reaches version 1.0.

### Cli Usage

    go install github.com/ajankovic/xdiff/cmd/xdiff
    xdiff -original original.xml -edited edited.xml

### Lib Usage

    import "github.com/ajankovic/xdiff"

	oFile, err := os.Open(originalFname)
	if err != nil {
        // handle error
	}
	eFile, err := os.Open(editedFname)
	if err != nil {
        // handle error
	}
    // Run compare algorithm on opened files.
	diff, err := xdiff.Compare(oFile, eFile)
	if err != nil {
        // handle error
	}
    // Output diff in plain text.
	enc := xdiff.PlainTextEncoder(os.Stdout)
	if err := enc.Encode(diff); err != nil {
        // handle error
	}

[GoDoc]: https://godoc.org/github.com/ajankovic/xdiff
[GoDoc Badge]: https://godoc.org/github.com/ajankovic/xdiff?status.svg
[GoReportCard]: https://goreportcard.com/report/github.com/ajankovic/xdiff
[GoReportCard Badge]: https://goreportcard.com/badge/github.com/ajankovic/xdiff