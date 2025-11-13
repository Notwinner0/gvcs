package objects

import (
	"bytes"
	"errors"
	"strings"
)

// kvlmParse parses a key-value list with message format.
func kvlmParse(raw []byte) (map[string][]string, string, error) {
	kvlm := make(map[string][]string)
	var message string

	// Find the end of the key-value pairs (first blank line)
	endOfPairs := bytes.Index(raw, []byte("\n\n"))
	if endOfPairs == -1 {
		// Maybe there's no message
		endOfPairs = len(raw)
	}

	// The rest is the message
	message = string(raw[endOfPairs+2:])

	pairs := bytes.Split(raw[:endOfPairs], []byte("\n"))
	var lastKey string

	for _, pair := range pairs {
		if len(pair) == 0 {
			continue
		}

		if pair[0] == ' ' { // Continuation line
			if lastKey == "" {
				return nil, "", errors.New("invalid kvlm: continuation line with no key")
			}
			// Append to the last value of the last key
			lastValueIndex := len(kvlm[lastKey]) - 1
			kvlm[lastKey][lastValueIndex] = kvlm[lastKey][lastValueIndex] + string(pair[1:])
		} else {
			space := bytes.Index(pair, []byte(" "))
			if space == -1 {
				return nil, "", errors.New("invalid kvlm: missing space")
			}
			key := string(pair[:space])
			value := string(pair[space+1:])
			kvlm[key] = append(kvlm[key], value)
			lastKey = key
		}
	}

	return kvlm, message, nil
}

// kvlmSerialize serializes a key-value map and a message back into bytes.
func kvlmSerialize(kvlm map[string][]string, message string) []byte {
	var b bytes.Buffer

	// Canonical order
	order := []string{"tree", "parent", "author", "committer", "gpgsig"}

	for _, key := range order {
		if values, ok := kvlm[key]; ok {
			for _, value := range values {
				// Handle multi-line values by prepending a space to subsequent lines
				valLines := strings.Split(value, "\n")
				b.WriteString(key)
				b.WriteString(" ")
				b.WriteString(valLines[0])
				b.WriteString("\n")
				for _, line := range valLines[1:] {
					b.WriteString(" ")
					b.WriteString(line)
					b.WriteString("\n")
				}
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(message)

	return b.Bytes()
}
