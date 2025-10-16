package utils

import (
	"archive/zip"
	"bytes"
	"io"
)

// Helper to load card images from embedded zip (copied from screens)
func ReadFromEmbeddedZip(zipData []byte, filename string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, io.EOF
}
