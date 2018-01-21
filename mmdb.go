// Copyright (c) 2018, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found s the LICENSE file.

// Package mmdb is a Go library for downloading and updating
// MaxMind GeoLite2 databases.
//
// Functions will download tar archive, extract the database file from it
// to a provided file name, and save MD5 sum of tar archive in a file
// in the same directory as the database file. MD5 sum is used for checking
// if the database is updated on the next function call.
package mmdb

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// GeoLite2 download URLs.
var (
	geoLite2CityURL    = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz"
	geoLite2CountryURL = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz"
	geoLite2ASNURL     = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-ASN.tar.gz"
)

// GeoLite2 database filenames inside tar archives.
var (
	geoLite2CityFilename    = "GeoLite2-City.mmdb"
	geoLite2CountryFilename = "GeoLite2-Country.mmdb"
	geoLite2ASNFilename     = "GeoLite2-ASN.mmdb"
)

// UpdateGeoLite2Country downloads and updates a GeoLite2 Country database and saves it
// under filename. MD5 sum of the tar archive is saved in a file in the same directory
// for update checks.
func UpdateGeoLite2Country(ctx context.Context, filename string) (saved bool, err error) {
	return update(ctx, filename, geoLite2CountryFilename, geoLite2CountryURL)
}

// UpdateGeoLite2City downloads and updates a GeoLite2 City database and saves it
// under filename. MD5 sum of the tar archive is saved in a file in the same directory
// for update checks.
func UpdateGeoLite2City(ctx context.Context, filename string) (saved bool, err error) {
	return update(ctx, filename, geoLite2CityFilename, geoLite2CityURL)
}

// UpdateGeoLite2ASN downloads and updates a GeoLite2 ASN database and saves it
// under filename. MD5 sum of the tar archive is saved in a file in the same directory
// for update checks.
func UpdateGeoLite2ASN(ctx context.Context, filename string) (saved bool, err error) {
	return update(ctx, filename, geoLite2ASNFilename, geoLite2ASNURL)
}

func update(ctx context.Context, filename, dbname, url string) (saved bool, err error) {
	req, err := http.NewRequest(http.MethodGet, url+".md5", nil)
	if err != nil {
		return false, errors.WithMessage(err, "http request md5 file")
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, errors.WithMessage(err, "get md5 file")
	}
	defer r.Body.Close()

	md5, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return false, errors.WithMessage(err, "download md5 file")
	}
	md5 = bytes.TrimSpace(md5)

	md5Filename := filepath.Join(filepath.Dir(filename), req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:])

	if _, err := os.Stat(md5Filename); err == nil {
		md5Current, err := ioutil.ReadFile(md5Filename)
		if err != nil {
			return false, errors.WithMessage(err, "open md5 file")
		}
		md5Current = bytes.TrimSpace(md5Current)

		if bytes.Equal(md5, md5Current) {
			return false, nil
		}
	}
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, errors.WithMessage(err, "http request")
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	r, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, errors.WithMessage(err, "get tar")
	}
	defer r.Body.Close()

	gzr, err := gzip.NewReader(r.Body)
	if err != nil {
		return false, errors.WithMessage(err, "gzip reader")
	}

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, errors.WithMessage(err, "read tar")
		}
		if strings.HasSuffix(header.Name, dbname) {
			if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
				return false, errors.WithMessage(err, "create directory")
			}
			writer, err := os.Create(filename)
			if err != nil {
				return false, errors.WithMessage(err, "create db file")
			}
			_, err = io.Copy(writer, tr)
			err = errors.WithMessage(err, "write db file")
			writer.Close()
			if err == nil {
				saved = true
			}
			break
		}
	}

	if saved {
		if err := ioutil.WriteFile(md5Filename, md5, 0666); err != nil {
			return false, errors.WithMessage(err, "write md5 file")
		}
		if setTestM5Filename != nil {
			setTestM5Filename(md5Filename)
		}
	}

	return saved, err
}

var (
	testMD5Filename   string
	setTestM5Filename func(md5Filename string)
)
