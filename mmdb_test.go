// Copyright (c) 2018, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found s the LICENSE file.
package mmdb

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func init() {
	setTestM5Filename = func(md5Filename string) {
		testMD5Filename = md5Filename
	}
}

func TestUpdateGeoLite2Country(t *testing.T) {
	testUpdate(t, UpdateGeoLite2Country)
}

func TestUpdateGeoLite2City(t *testing.T) {
	testUpdate(t, UpdateGeoLite2City)
}

func TestUpdateGeoLite2ASN(t *testing.T) {
	testUpdate(t, UpdateGeoLite2ASN)
}

func testUpdate(t *testing.T, f func(ctx context.Context, filename string) (saved bool, err error)) {
	dir, err := ioutil.TempDir("", "mmdb_"+t.Name())
	if err != nil {
		t.Fatal(err)
	}
	file, err := ioutil.TempFile(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	filename := file.Name()
	defer os.RemoveAll(dir)

	// download a new file
	saved, err := f(context.Background(), filename)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Error("expected file to be saved, but it is not")
	}

	if testMD5Filename == "" {
		t.Error("expected testMD5Filename to be sat, but it is not")
	}

	fileStat, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}

	md5FileStat, err := os.Stat(testMD5Filename)
	if err != nil {
		t.Fatal(err)
	}

	// do not download a new file
	saved, err = f(context.Background(), filename)
	if err != nil {
		t.Fatal(err)
	}
	if saved {
		t.Error("expected file not to be saved, but it is")
	}

	newFileStat, err := os.Stat(filename)
	if err != nil {
		t.Fatal(err)
	}
	newMD5FileStat, err := os.Stat(testMD5Filename)
	if err != nil {
		t.Fatal(err)
	}

	if !fileStat.ModTime().Equal(newFileStat.ModTime()) {
		t.Error("expected file not to be changed, but it is")
	}
	if !md5FileStat.ModTime().Equal(newMD5FileStat.ModTime()) {
		t.Error("expected file not to be changed, but it is")
	}

	fileHash, err := fileMD5(filename)
	if err != nil {
		t.Fatal(err)
	}
	md5Hash, err := fileMD5(testMD5Filename)
	if err != nil {
		t.Fatal(err)
	}

	// simulate update by changing saved files
	if err := ioutil.WriteFile(filename, []byte("data"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(testMD5Filename, []byte("hash"), 0666); err != nil {
		t.Fatal(err)
	}

	// update
	saved, err = f(context.Background(), filename)
	if err != nil {
		t.Fatal(err)
	}
	if !saved {
		t.Error("expected file to be saved, but it is not")
	}

	if testMD5Filename == "" {
		t.Error("expected testMD5Filename to be sat, but it is not")
	}

	newFileHash, err := fileMD5(filename)
	if err != nil {
		t.Fatal(err)
	}
	newMD5Hash, err := fileMD5(testMD5Filename)
	if err != nil {
		t.Fatal(err)
	}

	if fileHash != newFileHash {
		t.Error("file hash and updated file hash are not the same")
	}
	if md5Hash != newMD5Hash {
		t.Error("md5 file hash and updated md5 file hash are not the same")
	}
}

func fileMD5(filename string) (hash string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := md5.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
