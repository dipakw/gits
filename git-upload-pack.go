package gits

import (
	"io"
)

// UploadPack handles the request phase and returns bytes to send back.
func (repo *Repo) UploadPack(r io.Reader, w io.Writer, cb func()) error {
	n, err := repo.Negotiate(r, w)

	if err != nil {
		return err
	}

	objects, err := repo.Traverse(n)

	if err != nil {
		return err
	}

	if cb != nil {
		cb()
	}

	return repo.Pack(objects, w)
}
