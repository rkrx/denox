package utils

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/axetroy/denox/internal/fs"
	"github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
)

// Download file from URL to the filepath
func DownloadFile(filepath string, url string) error {
	tmpl := fmt.Sprintf(`{{string . "prefix"}}{{ green "%s" }} {{counters . }} {{ bar . "[" "=" ">" "-" "]"}} {{percent . }} {{speed . }}{{string . "suffix"}}`, filepath)

	// Get the data
	response, err := http.Get(url)

	if err != nil {
		return errors.Wrapf(err, "Download `%s` fail", url)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return errors.New(fmt.Sprintf("download file with status code %d", response.StatusCode))
	}

	defer response.Body.Close()

	// Create the file
	writer, err := os.Create(filepath)

	if err != nil {
		return errors.Wrapf(err, "Create `%s` fail", filepath)
	}

	defer func() {
		err = writer.Close()

		if err != nil {
			_ = os.Remove(filepath)
		}
	}()

	bar := pb.ProgressBarTemplate(tmpl).Start64(response.ContentLength)

	bar.SetWriter(os.Stdout)

	barReader := bar.NewProxyReader(response.Body)

	_, err = io.Copy(writer, barReader)

	bar.Finish()

	if err != nil {
		err = errors.Wrap(err, "copy fail")
	}

	return err
}

// Decompress gzip file and return filepath
func Decompress(tarFile, dest string) (*string, error) {
	fileReader, err := os.Open(tarFile)

	if err != nil {
		return nil, errors.Wrapf(err, "open file `%s` fail", tarFile)
	}

	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)

	if err != nil {
		return nil, errors.Wrapf(err, "gzip decode fail")
	}

	defer gzipReader.Close()

	newFilepath := path.Join(dest, path.Base(strings.TrimSuffix(tarFile, ".gz")))

	// if file have exist. then remove it first
	if fs.PathExists(newFilepath) {
		if err = os.Remove(newFilepath); err != nil {
			return nil, errors.Wrapf(err, "remove file `%s` fail", newFilepath)
		}
	}

	fileWriter, err := os.Create(newFilepath)

	if err != nil {
		return nil, errors.Wrapf(err, "create file `%s` fail", newFilepath)
	}

	defer func() {
		err = fileWriter.Close()

		if err != nil {
			err = os.Remove(newFilepath)
		}
	}()

	if _, err = io.Copy(fileWriter, gzipReader); err != nil {
		return nil, err
	}

	if err := fileWriter.Chmod(os.FileMode(0x755)); err != nil {
		return nil, errors.Wrap(err, "change file mod fail")
	}

	return &newFilepath, nil
}