# X-Diff algorithm implemented in Go

[![GoDoc Badge]][GoDoc] [![GoReportCard Badge]][GoReportCard] [![Build Status](https://travis-ci.com/ajankovic/xdiff.svg?branch=master)](https://travis-ci.com/ajankovic/xdiff)

This project should be considered as __WORK IN PROGRESS__. Breaking changes will most likely happen the API so refrain from using it for anything important until it reaches the stable version. X-Diff algorithm paper can be found [here](http://pages.cs.wisc.edu/~yuanwang/papers/xdiff.pdf).

## Background

By the official [XML specification](https://www.w3.org/TR/REC-xml/), XML tree model is ordered. In practice however, it's up to the users to determine how they want their data to be interpreted. So for most cases XML can be represented as unordered tree model.

> Using an unordered model, change detection is substantially harder than using the ordered model, but the change result that it generates is more accurate.

X-Diff is an algorithm that proposes change detection between XML documents by treating them as unordered trees. The general problem becomes NP-hard, but by reducing the possible mappings between the documents X-Diff algorithm theoretically solves the problem in polynomial time. More precisely:

> *O(|T<sub>1</sub>| \* |T<sub>2</sub>| \* max{ deg(T<sub>1</sub>), deg(T<sub>2</sub>) } \* log<sub>2</sub>( max{  deg(T<sub>1</sub>), deg(T<sub>2</sub>) }))*

Where *|T<sub>1</sub>|* and *|T<sub>2</sub>|* denote number of nodes in trees T<sub>1</sub> and T<sub>2</sub>, and *max{deg(T<sub>1</sub>), deg(T<sub>1</sub>)}* denote maximum child nodes between the trees.

The core proposal and analysis of the paper are solid, but overall quality of the paper is not so good.

For example, proposed *Figure 4.2 Matching Algorithm* is flawed because it doesn't count for differences where tree levels are different. Also there are vague and unclear statements in the descriptions so lot is left to the interpretation (or correction) by the implementer.

## About this implementation

This is a hobby project. The goal, at first, was to implement XML diffing by just following the mentioned paper and doing what's said there verbatim. But as I started understanding (and falling in love with) the problem more I have expanded the design so it's not *actually* a verbatim X-Diff implementation.

The core of the design mentioned in the paper will always be part of the project:

1. Create XTree with node signatures and hashes.
2. Match trees using minimum cost maximum flow and minimum cost matching.
3. Generate edit script from minimum cost matching.

Additions I intend to add are beyond the goal of just implementing an algorithm:

1. Implement custom XML parser to efficiently generate XTree. Using standard XML parser is very slow. Reasons for this are multiple, from creating copies of underling buffer to the way it's handling unicode. In my initial tests it takes over 20 seconds to process document of ~850 MB. With some tradeoffs, custom in-place parser brings this down to ~5 seconds. Improving this farther along with adding robustness to the parser will be the goal of the project in the future.

2. Add support for diffing directories of XML documents. Guess what else can be represented as an unordered tree, directories and files. The idea is to allow parsing of entire directories containing XML documents and representing them as part of the diffing tree. That way edit script will show operations on directories as well. For example this is useful in detecting changes to Salesforce metadata which is just bunch of XML files in directories.

3. Improve upon proposed design by the paper to improve performance of the algorithm. For example cited paper is vague about defining and describing how to reduce matching space before main matching of the algorithm. Goal of the project will be to define and implement those parts of the design.

4. Provide both library and CLI application for diffing XML documents.

## Installation and Usage

The project is currently not stable enough so there are no official releases. Once design is set those will come.

For now you can use standard Go tooling to install the module:

    go install github.com/ajankovic/xdiff/cmd/xdiff
    // or
    go get github.com/ajankovic/xdiff

### CLI Usage

    xdiff -left original.xml -right edited.xml

### Library Usage

For a more elaborate example please check the _cmd/xdiff/main.go_.

    import (
        "github.com/ajankovic/xdiff"
    )

    // Instantiate custom parser.
    p := parser.New()
    // Parse the XTree.
    left, err := p.ParseFile(leftSource)
    if err != nil {
    	// handle error
    }
    right, err := p.ParseFile(rightSource)
    if err != nil {
    	// handle error
    }
    // Run compare algorithm on parsed trees.
    diff, err := xdiff.Compare(left, right)
    if err != nil {
    	// handle error
    }
    // Output diff in plain text to the STDOUT.
    enc := xdiff.PlainTextEncoder(os.Stdout)
    if err := enc.Encode(diff); err != nil {
    	// handle error
    }

## Author and Attribution

Owner: Aleksandar Janković (office@ajankovic.com)

Attribution:

- [Inspiration for the xml parser](https://github.com/robfordww/runxml)
- [More inspiration for the xml parser](http://www.aosabook.org/en/posa/parsing-xml-at-the-speed-of-light.html)

[GoDoc]: https://godoc.org/github.com/ajankovic/xdiff
[GoDoc Badge]: https://godoc.org/github.com/ajankovic/xdiff?status.svg
[GoReportCard]: https://goreportcard.com/report/github.com/ajankovic/xdiff
[GoReportCard Badge]: https://goreportcard.com/badge/github.com/ajankovic/xdiff