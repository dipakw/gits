package gits

import (
	"crypto/sha1"
	"encoding/binary"
	"io"
)

func (r *Repo) Pack(hashes map[string]bool, w io.Writer) error {
	// First line (negotiation result)
	if _, err := io.WriteString(w, "0008NAK\n"); err != nil {
		return err
	}

	// We'll hash as we write so we don't need to keep pack content in memory.
	h := sha1.New()
	mw := io.MultiWriter(w, h) // writes to w and updates hash

	// PACK header
	if _, err := mw.Write([]byte("PACK")); err != nil {
		return err
	}

	// Version (2)
	if err := binary.Write(mw, binary.BigEndian, uint32(2)); err != nil {
		return err
	}

	// Object count
	if err := binary.Write(mw, binary.BigEndian, uint32(len(hashes))); err != nil {
		return err
	}

	// Stream each object
	for hash, include := range hashes {
		if !include {
			continue
		}

		object, err := r.Object(hash)

		if err != nil {
			return err
		}

		header, err := object.Header()

		if err != nil {
			return err
		}

		if _, err := mw.Write(header); err != nil {
			return err
		}

		zContent, err := Zlib.Compress(object.Data)

		if err != nil {
			return err
		}

		if _, err := mw.Write(zContent); err != nil {
			return err
		}
	}

	// Append the SHA-1 trailer (computed from h)
	if _, err := w.Write(h.Sum(nil)); err != nil {
		return err
	}

	return nil
}
