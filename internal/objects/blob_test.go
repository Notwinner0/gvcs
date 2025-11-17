package objects

import "testing"

func TestGitBlob_Type(t *testing.T) {
    blob := &GitBlob{}
    expected := "blob"
    if blob.Type() != expected {
        t.Errorf("Expected %q, got %q", expected, blob.Type())
    }
}
