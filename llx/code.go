package llx

import (
	"encoding/base64"

	"golang.org/x/crypto/blake2b"
)

// AddChunk to the list of chunks
func (l *Code) AddChunk(c *Chunk) {
	l.Checksums[int32(len(l.Code))] = c.Checksum(l.Checksums)
	l.Code = append(l.Code, c)
}

// RemoveLastChunk from the current code
func (l *Code) RemoveLastChunk() {
	l.Code = l.Code[:len(l.Code)-1]
}

// ChunkIndex is the index of the last chunk that was added
func (l *Code) ChunkIndex() int32 {
	return int32(len(l.Code))
}

// LastChunk is the last chunk in the list or nil
func (l *Code) LastChunk() *Chunk {
	tl := len(l.Code)
	if tl == 0 {
		return nil
	}
	return l.Code[tl-1]
}

// checksum from this code
func (l *Code) checksum() string {
	originalID := l.Id
	l.Id = ""

	data, err := l.Marshal()
	if err != nil {
		panic("Failed to marshal LLX code for checksum calculation. Critical failure.")
	}

	l.Id = originalID

	hash := blake2b.Sum512(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// UpdateID of the piece of code
func (l *Code) UpdateID() {
	l.Id = l.checksum()
}

// AddSuggestion to CodeBundle
func (l *CodeBundle) AddSuggestion(msg string) {
	l.Suggestions = append(l.Suggestions, msg)
}
