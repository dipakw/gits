package gits

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

func (repo *Repo) Advertise(r io.Reader, w io.Writer, service string, cb func()) ([]byte, error) {
	if service != "git-upload-pack" && service != "git-receive-pack" {
		return nil, fmt.Errorf("unsupported service: %s", service)
	}

	var refs = map[string][]int{}
	var err error

	refsPath := repo.absPath("refs")
	refsStat := repo.fs.Stat(refsPath)

	if refsStat[0] == 2 {
		refs, err = repo.fs.Scan(refsPath, FS_TYPE_FILE, -1)

		if err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer

	// Write service header.
	buf.Write(pktLine(fmt.Sprintf("# service=%s\n", service)))

	// Write flush.
	buf.Write([]byte("0000"))

	// Write head.
	head, err := repo.getHead()

	if err != nil {
		return nil, err
	}

	beforeNull := fmt.Sprintf("%s %s", head.Hash, ternary(head.NoHead, "", "HEAD"))
	afterNull := strings.Join(ADVERTISE_CAPS, " ")

	if !head.NoHead && !head.Detached && head.Ref != "" {
		afterNull = fmt.Sprintf("%s symref=HEAD:%s", afterNull, head.Ref)
	}

	line := fmt.Sprintf("%s%c%s", beforeNull, 0, afterNull)
	buf.Write(pktLine(line))

	// Write refs.
	// refname = /<dir>/refs/<refname>
	for refname := range refs {
		hash, err := repo.fs.ReadFile(refname)

		if err != nil {
			return nil, err
		}

		advRef := fmt.Sprintf("refs%s", refname[len(refsPath):])
		buf.Write(pktLine(fmt.Sprintf("%s %s\n", strings.Trim(string(hash), "\n"), advRef)))
	}

	// Write flush.
	buf.Write([]byte("0000"))

	if cb != nil {
		cb()
	}

	if w != nil {
		if _, err := w.Write(buf.Bytes()); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (r *Repo) getHead() (*Head, error) {
	headFile := r.absPath("HEAD")
	headStat := r.fs.Stat(headFile)

	if headStat[0] != 1 {
		head := &Head{
			NoHead: true,
			Hash:   hex.EncodeToString(make([]byte, 20)),
		}

		return head, nil
	}

	headBytes, err := r.fs.ReadFile(headFile)

	if err != nil {
		return nil, err
	}

	head := &Head{}
	solved := false
	headStr := strings.TrimSpace(string(headBytes))

	// 1. First test if the head is a ref.
	if strings.HasPrefix(headStr, "ref: ") {
		head.Ref = strings.TrimPrefix(headStr, "ref: ")

		refFile := r.absPath(head.Ref)
		stat := r.fs.Stat(refFile)

		// File not found, or size is 0.
		if stat[0] == 0 || stat[1] == 0 {
			head.Unborn = true
			head.Hash = hex.EncodeToString(make([]byte, 20))
		}

		// File found
		if stat[0] == 1 {
			refHash, err := r.fs.ReadFile(refFile)

			if err != nil {
				return nil, err
			}

			head.Hash = strings.TrimSpace(string(refHash))
		}

		solved = true
	}

	// TODO: Support worktrees.
	// 2. gitdir: ../.git/worktrees/feature-branch

	// 3. Detached head.
	if !solved {
		_, err = hex.DecodeString(headStr)

		if err == nil {
			head.Detached = true
		}

		solved = true
	}

	if !solved {
		return nil, fmt.Errorf("could not determine HEAD")
	}

	return head, nil
}
