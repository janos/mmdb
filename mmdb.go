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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// GeoLite2 download URLs.
var (
	geoLite2CityURL    = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City"
	geoLite2CountryURL = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country"
	geoLite2ASNURL     = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN"
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
func UpdateGeoLite2Country(ctx context.Context, filename, licenseKey string) (saved bool, err error) {
	return update(ctx, filename, geoLite2CountryFilename, geoLite2CountryURL, licenseKey)
}

// UpdateGeoLite2City downloads and updates a GeoLite2 City database and saves it
// under filename. MD5 sum of the tar archive is saved in a file in the same directory
// for update checks.
func UpdateGeoLite2City(ctx context.Context, filename, licenseKey string) (saved bool, err error) {
	return update(ctx, filename, geoLite2CityFilename, geoLite2CityURL, licenseKey)
}

// UpdateGeoLite2ASN downloads and updates a GeoLite2 ASN database and saves it
// under filename. MD5 sum of the tar archive is saved in a file in the same directory
// for update checks.
func UpdateGeoLite2ASN(ctx context.Context, filename, licenseKey string) (saved bool, err error) {
	return update(ctx, filename, geoLite2ASNFilename, geoLite2ASNURL, licenseKey)
}

func update(ctx context.Context, filename, dbname, address, licenseKey string) (saved bool, err error) {
	u, err := url.Parse(address)
	if err != nil {
		return false, err
	}
	q := u.Query()
	q.Set("license_key", licenseKey)
	q.Set("suffix", "tar.gz.md5")
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("http request md5 file: %w", err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("get md5 file: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected http response %s", r.Status)
	}

	md5, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("download md5 file: %w", err)
	}
	md5 = bytes.TrimSpace(md5)

	md5Filename := filepath.Join(filepath.Dir(filename), req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:])

	if _, err := os.Stat(md5Filename); err == nil {
		md5Current, err := ioutil.ReadFile(md5Filename)
		if err != nil {
			return false, fmt.Errorf("open md5 file: %w", err)
		}
		md5Current = bytes.TrimSpace(md5Current)

		if bytes.Equal(md5, md5Current) {
			return false, nil
		}
	}
	q.Set("suffix", "tar.gz")
	u.RawQuery = q.Encode()
	req, err = http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("http request: %w", err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	r, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("get tar: %w", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected http response %s", r.Status)
	}

	gzr, err := gzip.NewReader(r.Body)
	if err != nil {
		return false, fmt.Errorf("gzip reader: %w", err)
	}

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return false, fmt.Errorf("read tar: %w", err)
		}
		if strings.HasSuffix(header.Name, "/"+dbname) {
			if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
				return false, fmt.Errorf("create directory: %w", err)
			}
			writer, err := os.Create(filename)
			if err != nil {
				return false, fmt.Errorf("create db file: %w", err)
			}
			_, err = io.Copy(writer, tr)
			_ = writer.Close()
			if err != nil {
				return false, fmt.Errorf("write db file: %w", err)
			}
			saved = true
			break
		}
	}

	if saved {
		if err := ioutil.WriteFile(md5Filename, md5, 0666); err != nil {
			return false, fmt.Errorf("write md5 file: %w", err)
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
