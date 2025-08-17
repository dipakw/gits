package gits

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

/*
 * ----- input -----
 * tree xxx
 * parent xxx
 * parent xxx
 *
 * ---- return ----
 * { tree: [xxx], parent: [xxx, xxx] }
 */
func parseLinesKV(data []byte) map[string][]string {
	lines := strings.Split(string(data), "\n")

	kv := make(map[string][]string)

	for _, line := range lines {
		if line == "" {
			break
		}

		parts := strings.SplitN(line, " ", 2)

		if len(parts) < 2 {
			continue
		}

		key, value := parts[0], parts[1]

		if _, ok := kv[key]; !ok {
			kv[key] = []string{}
		}

		kv[key] = append(kv[key], value)
	}

	return kv
}

/*
 * Reads a single Git pkt-line from br.
 *
 * Returns:
 *   - data: line payload (no length prefix)
 *   - flush: true if this was a flush packet (0000)
 *   - err: error or nil
 */
func readPktLine(br *bufio.Reader) (data string, flush bool, err error) {
	// Read 4-byte ASCII hex length (e.g., "0032" or "0000")
	lenBytes := make([]byte, 4)

	if _, err = io.ReadFull(br, lenBytes); err != nil {
		return "", false, err
	}

	if string(lenBytes) == "0000" {
		// Flush packet
		return "", true, nil
	}

	// Convert ASCII hex to uint16 length
	rawLen := make([]byte, 2)

	if _, err = hex.Decode(rawLen, lenBytes); err != nil {
		return "", false, fmt.Errorf("bad length prefix: %q", lenBytes)
	}

	size := int(rawLen[0])<<8 + int(rawLen[1])

	if size < 4 {
		return "", false, fmt.Errorf("invalid pkt-line length: %d", size)
	}

	// Read the remaining data (length minus the 4-byte prefix)
	dataBytes := make([]byte, size-4)

	if _, err = io.ReadFull(br, dataBytes); err != nil {
		return "", false, err
	}

	return strings.TrimSpace(string(dataBytes)), false, nil
}

// pktLine encodes a non-empty pkt-line; use "0000" for flush.
func pktLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}

func getPackObjectHeader(br *bufio.Reader) (uint8, uint64, error) {
	var typ uint8
	var size uint64

	// --- First byte ---
	firstByte, err := br.ReadByte()

	if err != nil {
		return 0, 0, err
	}

	typ = uint8((firstByte >> 4) & 0x07) // bits 6–4 = type
	size = uint64(firstByte & 0x0F)      // low 4 bits of size
	shift := uint(4)                     // already took 4 bits

	// --- Continuation bytes ---
	if firstByte&0x80 != 0 { // bit 7 = continuation
		for {
			b, err := br.ReadByte()

			if err != nil {
				return 0, 0, err
			}

			size |= uint64(b&0x7F) << shift
			shift += 7
			if b&0x80 == 0 { // no continuation
				break
			}
		}
	}

	return typ, size, nil
}

func getPackObjectContent(br *bufio.Reader, typ uint8, size uint64) ([]byte, []byte, error) {
	var base []byte = nil

	// Skip delta metadata if needed
	switch typ {
	case OBJ_OFS_DELTA:
		for {
			b, err := br.ReadByte()

			if err != nil {
				return nil, nil, err
			}

			if b&0x80 == 0 { // last byte
				break
			}
		}

	case OBJ_REF_DELTA:
		base = make([]byte, 20)

		if _, err := io.ReadFull(br, base); err != nil {
			return nil, nil, err
		}
	}

	buf, err := Zlib.Inflate(br, size)

	if err != nil {
		return nil, nil, err
	}

	return buf, base, nil
}

func readSize(r *bytes.Reader) (uint64, error) {
	var result uint64
	var shift uint

	for {
		b, err := r.ReadByte()

		if err != nil {
			return 0, err
		}

		result |= uint64(b&0x7F) << shift

		if b&0x80 == 0 {
			break
		}

		shift += 7
	}

	return result, nil
}

func parseDeltaOps(r *bytes.Reader) ([]*DeltaOp, error) {
	var ops []*DeltaOp

	for {
		cmd, err := r.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read command: %w", err)
		}

		if cmd&0x80 != 0 {
			// Copy from base
			var offset, size uint64
			if cmd&0x01 != 0 {
				b, _ := r.ReadByte()
				offset |= uint64(b)
			}
			if cmd&0x02 != 0 {
				b, _ := r.ReadByte()
				offset |= uint64(b) << 8
			}
			if cmd&0x04 != 0 {
				b, _ := r.ReadByte()
				offset |= uint64(b) << 16
			}
			if cmd&0x08 != 0 {
				b, _ := r.ReadByte()
				offset |= uint64(b) << 24
			}
			if cmd&0x10 != 0 {
				b, _ := r.ReadByte()
				size |= uint64(b)
			}
			if cmd&0x20 != 0 {
				b, _ := r.ReadByte()
				size |= uint64(b) << 8
			}
			if cmd&0x40 != 0 {
				b, _ := r.ReadByte()
				size |= uint64(b) << 16
			}
			if size == 0 {
				size = 0x10000
			}
			ops = append(ops, &DeltaOp{
				Copy:   true,
				Offset: offset,
				Size:   size,
			})
		} else {
			// Insert literal
			size := uint64(cmd & 0x7F)
			data := make([]byte, size)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, fmt.Errorf("read literal: %w", err)
			}
			ops = append(ops, &DeltaOp{
				Copy: false,
				Size: size,
				Data: data,
			})
		}
	}

	return ops, nil
}

//	Example refs: [][]string{
//			[]string{"refs/heads/main", "aaa", "bbb"},
//			[]string{"refs/heads/master", "ccc", "ddd"},
//	}
func prepSuccessRes(refs [][]string) []byte {
	var buf bytes.Buffer

	// Write "unpack ok\n" to indicate successful packfile unpacking
	buf.Write(pktLine("unpack ok\n"))

	// Write "ok <ref>\n" for each reference
	for _, ref := range refs {
		buf.Write(pktLine("ok " + ref[0] + "\n"))
	}

	// Write flush packet
	buf.WriteString("0000")

	return buf.Bytes()
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}

	return b
}
