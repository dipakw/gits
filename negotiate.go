package gits

import (
	"bufio"
	"io"
	"strings"
)

func (repo *Repo) Negotiate(r io.Reader, w io.Writer) (*Negotiation, error) {
	var n = &Negotiation{
		Wants: map[string]bool{},
		Haves: map[string]bool{},
		Done:  false,
		EOF:   false,
	}

	br := bufio.NewReader(r)

	// Read wants.
	for {
		line, flush, err := readPktLine(br)

		if err != nil {
			if err == io.EOF {
				n.EOF = true
				break
			}

			return nil, err
		}

		if flush {
			break
		}

		if line == "" {
			continue
		}

		if line == "done" {
			n.Done = true
			break
		}

		if strings.HasPrefix(line, "want ") {
			parts := strings.SplitN(line, " ", 3)
			n.Wants[parts[1]] = true
		}
	}

	// Return if done.
	if n.Done {
		return n, nil
	}

	// TOOD: Read haves.

	return n, nil
}
