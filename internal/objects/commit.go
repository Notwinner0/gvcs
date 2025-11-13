package objects

// GitCommit represents a commit object.
type GitCommit struct {
	Kvlm    map[string][]string
	Message string
}

func (c *GitCommit) Type() string {
	return "commit"
}

func (c *GitCommit) Deserialize(data []byte) error {
	var err error
	c.Kvlm, c.Message, err = kvlmParse(data)
	return err
}

func (c *GitCommit) Serialize() ([]byte, error) {
	return kvlmSerialize(c.Kvlm, c.Message), nil
}
