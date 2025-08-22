package gits

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

func (repo *Repo) ReceivePack(r io.Reader, w io.Writer, cb func()) error {
	br := bufio.NewReader(r)

	// Refs to be upated.
	// Ref name, Old hash, New hash
	refs := [][]string{}

	for {
		line, flush, err := readPktLine(br)

		if err != nil {
			return err
		}

		if flush {
			break
		}

		parts := strings.Split(line, " ")

		if len(parts) < 3 {
			return errors.New("invalid ref update line: " + line)
		}

		refs = append(refs, []string{
			parts[2], // Ref name, e.g. refs/heads/main
			parts[0], // Old hash
			parts[1], // New hash
		})
	}

	if err := repo.Unpack(br); err != nil {
		return err
	}

	for _, ref := range refs {
		repo.fs.WriteFile(repo.absPath(ref[0]), []byte(ref[2]))
	}

	res := prepSuccessRes(refs)

	if cb != nil {
		cb()
	}

	if _, err := w.Write(res); err != nil {
		return err
	}

	return nil
}
