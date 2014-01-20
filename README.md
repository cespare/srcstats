# srcstats

A source code analysis thing.

## Install

    go get github.com/cespare/srcstats

## Usage

    $ srcstats -h
    Usage: srcstats [OPTIONS] FILE1 FILE2 ... (or pass filenames from stdin)
    where OPTIONS are:
      -tabwidth=4: Width to assign tabs for determining line length

## Examples

Just give it some files:

    $ srcstats srcstats.go
    files                                  1
    total size                         4.6KB
    mean file size                     4.6KB
    total lines                          211
    lines / file                       211.0
    non-empty lines                      192
    non-empty lines / file             192.0
    chars / non-empty line              19.4
    mean non-empty line length          28.6

It reads from stdin as well:

    $ find $GOROOT/src -name '*.go' | ./srcstats -tabwidth=8
    files                               1355
    total size                          11MB
    mean file size                     8.0KB
    total lines                       403359
    lines / file                       297.7
    non-empty lines                   365354
    non-empty lines / file             269.6
    chars / non-empty line              23.5
    mean non-empty line length          38.6

It's pretty snappy; here's srcstats analyzing the Linux source tree:

    $ find $REPOS/linux -name '*.c' -o -name '*.h' | /usr/bin/time srcstats
    files                              35406
    total size                        425MiB
    mean file size                     12KiB
    total lines                     16322358
    lines / file                       461.0
    non-empty lines                 14086746
    non-empty lines / file             397.9
    chars / non-empty line              26.9
    mean non-empty line length          35.5
    srcstats  9.07s user 0.44s system 256% cpu 3.699 total
