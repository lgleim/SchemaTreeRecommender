package io

import (
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/biogo/hts/bgzf"
	gzip "github.com/klauspost/pgzip"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// UniversalReader opens a file for read mode. It is able to automatically decompress
// files that use either GZ, BGZ or BZ2. It will also display a loading bar on stdout.
func UniversalReader(fileName string) (reader io.ReadCloser, err error) {
	if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
		fmt.Println("Reading data from stdin")
		reader = os.Stdin
	} else {
		var file *os.File
		file, err = os.Open(fileName)
		if err != nil {
			return
		}

		var stat os.FileInfo
		stat, err = file.Stat()
		if err != nil {
			return
		}
		if stat.IsDir() {
			err = errors.New("Reading entire directories is not yet possible")
			return
		}
		fmt.Printf("Reading data from file '%v'. Progress w.r.t. on-disk-size: \n", fileName)

		// create and start progress bar
		bar := pb.New(int(stat.Size())).SetUnits(pb.U_BYTES).SetRefreshRate(500 * time.Millisecond).Start()
		bar.ShowElapsedTime = true
		bar.ShowSpeed = true
		reader = bar.NewProxyReader(file)

		// decompress stream if applicable
		switch ext := filepath.Ext(fileName); ext {
		case ".bz2":
			reader = ioutil.NopCloser(bzip2.NewReader(reader)) // Decompression
		case ".gz":
			reader, err = gzip.NewReaderN(reader, 8*1024*1024, 48) // readahead
		case ".bgz":
			reader, err = bgzf.NewReader(reader, 0)
			//case ".zst", ".zstd":
			//reader = releaseCloser{gozstd.NewReader(reader)}
		}

		if err != nil {
			return
		}

		reader = finishCloser{reader, bar, file}
	}
	return
}

type finishCloser struct {
	io.ReadCloser
	bar  *pb.ProgressBar
	file *os.File
}

func (r finishCloser) Close() error {
	r.bar.Finish()
	defer r.file.Close()
	return r.ReadCloser.Close()
}

// GZipWriteCloser implements the necessary methods to write and make a cascading close of all
// underlying files.
type GZipWriteCloser struct {
	gzipHandle *gzip.Writer
	fileHandle *os.File
}

// Write writes a byte stream to the gzip writer.
func (gzwc GZipWriteCloser) Write(p []byte) (n int, err error) {
	return gzwc.gzipHandle.Write(p)
}

// Close will close the gzip writer and underlying file handle.
// TODO: There might be a more consise way to do this, but I was not sure if defer can actually
//       return something.
func (gzwc GZipWriteCloser) Close() (err error) {
	err = gzwc.gzipHandle.Close()
	if err != nil {
		return err
	}
	err = gzwc.fileHandle.Close()
	if err != nil {
		return err
	}
	return nil
}

// CreateAndOpenWithGzip opens a file with a GZip writer and returns an structure capable to closing
// all underlying open files.
func CreateAndOpenWithGzip(filePath string) io.WriteCloser {

	// Create the file handle
	fileh, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}

	// Create the GZip writer
	gzipw, err := gzip.NewWriterLevel(fileh, gzip.BestCompression)
	if err != nil {
		log.Fatal(err)
	}

	return GZipWriteCloser{gzipHandle: gzipw, fileHandle: fileh}
}

// TrimCompressionExtension will remove the extension of a fileName if it resembles
// an extension used by compression algorithms.
func TrimCompressionExtension(fileName string) (fileBase string) {
	ext := filepath.Ext(fileName)
	if ext == ".bz2" || ext == ".gz" || ext == ".gbz" {
		return strings.TrimSuffix(fileName, ext)
	}
	return fileName
}
