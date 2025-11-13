package objects

// GitBlob represents a blob object.
type GitBlob struct {
	data []byte
}

func (b *GitBlob) Serialize() ([]byte, error) {
	return b.data, nil
}

func (b *GitBlob) Deserialize(data []byte) error {
	b.data = data
	return nil
}

func (b *GitBlob) Type() string {
	return "blob"
}
