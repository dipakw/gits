package gits

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func (repo *Repo) Negotiate(r io.Reader, w io.Writer) (*Negotiation, error) {
	var n = &Negotiation{
		Wants: map[string]bool{},
		Haves: map[string]bool{},
		Caps:  map[string]bool{},
		Agent: "",
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
			parts := strings.Split(line, " ")

			if len(parts) < 2 {
				return nil, fmt.Errorf("invalid want line: %s", line)
			}

			n.Wants[parts[1]] = true

			if len(parts) > 2 {
				for _, cap := range parts[2:] {
					if strings.HasPrefix(cap, "agent=") {
						n.Agent = cap[6:]
						continue
					}

					n.Caps[cap] = true
				}
			}
		}
	}

	// Return if done.
	if n.Done {
		return n, nil
	}

	// Read haves.
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
			// Batch of haves.
			continue
		}

		if line == "" {
			continue
		}

		if line == "done" {
			n.Done = true
			break
		}

		if strings.HasPrefix(line, "have ") {
			parts := strings.SplitN(line, " ", 3)

			if len(parts) < 2 {
				return nil, fmt.Errorf("invalid have line: %s", line)
			}

			n.Haves[parts[1]] = true
		}
	}

	return n, nil
}
