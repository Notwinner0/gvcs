package objects

// GitTag represents a tag object.
type GitTag struct {
	Kvlm    map[string][]string
	Message string
}

func (t *GitTag) Type() string {
	return "tag"
}

func (t *GitTag) Deserialize(data []byte) error {
	var err error
	t.Kvlm, t.Message, err = kvlmParse(data)
	return err
}

func (t *GitTag) Serialize() ([]byte, error) {
	// The canonical order for tags is different
	// order := []string{"object", "type", "tag", "tagger", "gpgsig"}
	// We need a slightly modified serializer for tags if the order matters strictly.
	// For simplicity here, we'll reuse the commit serializer's logic.
	return kvlmSerialize(t.Kvlm, t.Message), nil
}
