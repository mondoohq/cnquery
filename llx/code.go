package llx

// AddChunk to the list of chunks
func (l *Code) AddChunk(c *Chunk) {
	l.Checksums[l.ChunkIndex()+1] = c.Checksum(l)
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
	return l.Checksums[l.ChunkIndex()]
}

// UpdateID of the piece of code
func (l *Code) UpdateID() {
	l.Id = l.checksum()
}

// AddSuggestion to CodeBundle
func (l *CodeBundle) AddSuggestion(msg string) {
	l.Suggestions = append(l.Suggestions, msg)
}
