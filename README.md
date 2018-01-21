# Go package for downloading MaxMind GeoLite2 database

[![GoDoc](https://godoc.org/resenje.org/mmdb?status.svg)](https://godoc.org/resenje.org/mmdb)
[![Build Status](https://travis-ci.org/janos/mmdb.svg?branch=master)](https://travis-ci.org/janos/mmdb)

Functions in this package are downloading MaxMind Geolite2 tar archives,
extracting the database file from it to a provided file name, and saving
MD5 sum of tar archive in a file in the same directory as the database file.
MD5 sum is used for checking if the database is updated on the next function
call.

## Installation

```sh
go get resenje.org/mmdb
```
