package browserstore

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const gzipPrefix = "gzip:"

// Encode compresses save data into a string suitable for localStorage.
func Encode(data []byte) (string, error) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		return "", fmt.Errorf("compress browser save: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("finish browser save compression: %w", err)
	}
	return gzipPrefix + base64.StdEncoding.EncodeToString(compressed.Bytes()), nil
}

// Decode restores browser save data and accepts legacy uncompressed JSON.
func Decode(value string) ([]byte, error) {
	if !strings.HasPrefix(value, gzipPrefix) {
		return []byte(value), nil
	}
	compressed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, gzipPrefix))
	if err != nil {
		return nil, fmt.Errorf("decode browser save: %w", err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("open browser save: %w", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("decompress browser save: %w", err)
	}
	return data, nil
}
