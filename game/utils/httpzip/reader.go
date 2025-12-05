package httpzip

import (
	"archive/zip"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"unsafe"
)

type Decompressor func(r io.Reader) io.ReadCloser

var decompressors = map[uint16]Decompressor{
	zip.Store:   func(r io.Reader) io.ReadCloser { return io.NopCloser(r) },
	zip.Deflate: flate.NewReader,
}

const (
	fileHeaderLen       = 30
	fileHeaderSignature = 0x04034b50
)

type Reader struct {
	url    string
	client *http.Client
	files  []*File
}

type File struct {
	*zip.File
	url        string
	client     *http.Client
	bodyOffset *int64
	mu         sync.Mutex
	readerAt   *httpReaderAt
}

type httpReaderAt struct {
	url    string
	client *http.Client
}

func (r *httpReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return 0, err
	}
	rangeHeader := fmt.Sprintf("bytes=%d-%d", off, off+int64(len(p))-1)
	req.Header.Set("Range", rangeHeader)

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	return io.ReadFull(resp.Body, p)
}

type httpRangeReader struct {
	url       string
	client    *http.Client
	offset    int64
	remaining int64
	buffer    []byte
	bufOffset int
}

func (r *httpRangeReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > r.bufOffset {
		n = copy(p, r.buffer[r.bufOffset:])
		r.bufOffset += n
		r.offset += int64(n)
		r.remaining -= int64(n)
		if n == len(p) || r.remaining <= 0 {
			return n, nil
		}
		p = p[n:]
	}

	if r.remaining <= 0 {
		return n, io.EOF
	}

	toRead := int64(len(p))
	if toRead > r.remaining {
		toRead = r.remaining
	}

	req, err := http.NewRequest("GET", r.url, nil)
	if err != nil {
		return n, err
	}
	rangeHeader := fmt.Sprintf("bytes=%d-%d", r.offset, r.offset+toRead-1)
	req.Header.Set("Range", rangeHeader)

	resp, err := r.client.Do(req)
	if err != nil {
		return n, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return n, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	readCount, err := io.ReadFull(resp.Body, p[:toRead])
	r.offset += int64(readCount)
	r.remaining -= int64(readCount)
	n += readCount
	return n, err
}

func NewReader(url string, client *http.Client) (*Reader, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Head(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return nil, errors.New("server doesn't accept ranges")
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, err
	}

	readerAt := &httpReaderAt{url: url, client: client}
	zr, err := zip.NewReader(readerAt, size)
	if err != nil {
		return nil, err
	}

	files := make([]*File, len(zr.File))
	for i, f := range zr.File {
		files[i] = &File{
			File:       f,
			url:        url,
			client:     client,
			bodyOffset: nil,
			readerAt:   readerAt,
		}
	}

	return &Reader{
		url:    url,
		client: client,
		files:  files,
	}, nil
}

func getHeaderOffset(f *zip.File) int64 {
	rv := reflect.ValueOf(f).Elem()
	field := rv.FieldByName("headerOffset")
	if !field.IsValid() {
		return 0
	}
	return *(*int64)(unsafe.Pointer(field.UnsafeAddr()))
}

func (f *File) getBodyOffsetAndPrefetch() (int64, []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	compressedSize := int64(f.CompressedSize64)
	if compressedSize == 0 {
		compressedSize = int64(f.CompressedSize64)
	}

	if f.bodyOffset != nil {
		return *f.bodyOffset, nil, nil
	}

	headerOffset := getHeaderOffset(f.File)

	estimatedHeaderSize := int64(fileHeaderLen + len(f.Name) + 1024)
	prefetchSize := estimatedHeaderSize + compressedSize
	if prefetchSize > 1024*1024 {
		prefetchSize = 1024 * 1024
	}

	prefetchBuf := make([]byte, prefetchSize)
	n, err := f.readerAt.ReadAt(prefetchBuf, headerOffset)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, nil, fmt.Errorf("failed to prefetch for %s: %w", f.Name, err)
	}
	prefetchBuf = prefetchBuf[:n]

	if len(prefetchBuf) < fileHeaderLen {
		return 0, nil, fmt.Errorf("prefetch buffer too small for %s", f.Name)
	}

	sig := binary.LittleEndian.Uint32(prefetchBuf[:4])
	if sig != fileHeaderSignature {
		return 0, nil, errors.New("invalid file header signature")
	}

	filenameLen := int(binary.LittleEndian.Uint16(prefetchBuf[26:28]))
	extraLen := int(binary.LittleEndian.Uint16(prefetchBuf[28:30]))
	bodyOffset := headerOffset + fileHeaderLen + int64(filenameLen) + int64(extraLen)

	f.bodyOffset = &bodyOffset

	headerSize := int(bodyOffset - headerOffset)
	if headerSize < len(prefetchBuf) {
		dataBuf := prefetchBuf[headerSize:]
		return bodyOffset, dataBuf, nil
	}

	return bodyOffset, nil, nil
}

func (f *File) Open() (io.ReadCloser, error) {
	bodyOffset, prefetchData, err := f.getBodyOffsetAndPrefetch()
	if err != nil {
		return nil, err
	}

	compressedSize := int64(f.CompressedSize64)
	if compressedSize == 0 {
		compressedSize = int64(f.CompressedSize64)
	}

	r := &httpRangeReader{
		url:       f.url,
		client:    f.client,
		offset:    bodyOffset + int64(len(prefetchData)),
		remaining: compressedSize - int64(len(prefetchData)),
		buffer:    prefetchData,
		bufOffset: 0,
	}

	decompressor := decompressors[f.Method]
	if decompressor == nil {
		return nil, fmt.Errorf("unsupported compression method: %d", f.Method)
	}

	if f.Method == zip.Store {
		return io.NopCloser(r), nil
	}

	return decompressor(r), nil
}

func (r *Reader) File() []*File {
	return r.files
}

func (r *Reader) ReadFile(name string) ([]byte, error) {
	var file *File
	for _, f := range r.files {
		if f.Name == name {
			file = f
			break
		}
	}
	if file == nil {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}
