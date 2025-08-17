package gits

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

func zlibCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := zlib.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		writer.Close() // still close on error
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func zlibDecompress(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)

	reader, err := zlib.NewReader(buf)

	if err != nil {
		return nil, err
	}

	defer reader.Close()

	return io.ReadAll(reader)
}

func zlibInflate(br *bufio.Reader, size uint64) ([]byte, error) {
	zr, err := zlib.NewReader(br)

	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}

	defer zr.Close()

	buf := make([]byte, size)
	n, err := io.ReadFull(zr, buf)

	if err != nil {
		return nil, err
	}

	if uint64(n) != size {
		return nil, fmt.Errorf("unexpected size: got %d, want %d", n, size)
	}

	return buf, nil
}

var Zlib = struct {
	Compress   func([]byte) ([]byte, error)
	Decompress func([]byte) ([]byte, error)
	Inflate    func(*bufio.Reader, uint64) ([]byte, error)
}{
	Compress:   zlibCompress,
	Decompress: zlibDecompress,
	Inflate:    zlibInflate,
}
