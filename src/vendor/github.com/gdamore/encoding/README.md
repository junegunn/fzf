## encoding

[![Linux Status](https://img.shields.io/travis/gdamore/encoding.svg?label=linux)](https://travis-ci.org/gdamore/encoding)
[![Windows Status](https://img.shields.io/appveyor/ci/gdamore/encoding.svg?label=windows)](https://ci.appveyor.com/project/gdamore/encoding)
[![Apache License](https://img.shields.io/badge/license-APACHE2-blue.svg)](https://github.com/gdamore/encoding/blob/master/LICENSE)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/gdamore/encoding)
[![Go Report Card](http://goreportcard.com/badge/gdamore/encoding)](http://goreportcard.com/report/gdamore/encoding)

Package encoding provides a number of encodings that are missing from the
standard Go [encoding]("https://godoc.org/golang.org/x/text/encoding") package.

We hope that we can contribute these to the standard Go library someday.  It
turns out that some of these are useful for dealing with I/O streams coming
from non-UTF friendly sources.

The UTF8 Encoder is also useful for situations where valid UTF-8 might be
carried in streams that contain non-valid UTF; in particular I use it for
helping me cope with terminals that embed escape sequences in otherwise
valid UTF-8.
