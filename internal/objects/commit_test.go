package objects

import (
	"reflect"
	"testing"
)

func TestGitCommit_Type(t *testing.T) {
	commit := &GitCommit{}
	expected := "commit"
	if commit.Type() != expected {
		t.Errorf("Expected %q, got %q", expected, commit.Type())
	}
}

func TestGitCommit_Deserialize(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    *GitCommit
		wantErr bool
	}{
		{
			name: "basic commit",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nInitial commit",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Initial commit",
			},
			wantErr: false,
		},
		{
			name: "merge commit with multiple parents",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nparent 1234567890abcdef1234567890abcdef12345678\nparent abcdef1234567890abcdef1234567890abcdef12\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nMerge branch 'feature'",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"parent":    {"1234567890abcdef1234567890abcdef12345678", "abcdef1234567890abcdef1234567890abcdef12"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Merge branch 'feature'",
			},
			wantErr: false,
		},
		{
			name: "commit with multi-line message",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nAdd feature X\n\nThis commit adds the new feature X that allows users to:\n- Do something useful\n- Do something else\n\nAlso fixes bug Y.",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Add feature X\n\nThis commit adds the new feature X that allows users to:\n- Do something useful\n- Do something else\n\nAlso fixes bug Y.",
			},
			wantErr: false,
		},
		{
			name: "commit with continuation lines (valid)",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\ngpgsig -----BEGIN PGP SIGNATURE-----\n iQIzBAABCAAdFiEEfake1234567890abcdefg=+0000FAKE\n =ABCD\n -----END PGP SIGNATURE-----\n\nSigned commit with GPG",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
					"gpgsig":    {"-----BEGIN PGP SIGNATURE-----iQIzBAABCAAdFiEEfake1234567890abcdefg=+0000FAKE=ABCD-----END PGP SIGNATURE-----"},
				},
				Message: "Signed commit with GPG",
			},
			wantErr: false,
		},
		{
			name: "commit with encoding field",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nencoding utf-8\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nCommit with encoding specified",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"encoding":  {"utf-8"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Commit with encoding specified",
			},
			wantErr: false,
		},
		{
			name: "commit with no message",
			data: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\n",
			want: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "",
			},
			wantErr: false,
		},
		{
			name:    "invalid kvlm - missing space",
			data:    "tree4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\n\nInitial commit",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commit := &GitCommit{}
			err := commit.Deserialize([]byte(tt.data))

			if (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if !reflect.DeepEqual(commit, tt.want) {
				t.Errorf("Deserialize() got = %+v, want %+v", commit, tt.want)
			}
		})
	}
}

func TestGitCommit_RoundTrip(t *testing.T) {
	// Test serialization and deserialization round trip
	original := &GitCommit{
		Kvlm: map[string][]string{
			"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
			"parent":    {"1234567890abcdef1234567890abcdef12345678", "abcdef1234567890abcdef1234567890abcdef12"},
			"author":    {"John Doe <john@example.com> 1234567890 +0000"},
			"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
		},
		Message: "Merge branch 'feature' with multi-line\n\nThis is a detailed message.",
	}

	// Serialize
	serialized, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	// Deserialize into new commit
	deserialized := &GitCommit{}
	err = deserialized.Deserialize(serialized)
	if err != nil {
		t.Fatalf("Deserialize() failed: %v", err)
	}

	// Should be equal to original
	if !reflect.DeepEqual(original, deserialized) {
		t.Errorf("Round trip failed:\nOriginal: %+v\nDeserialized: %+v", original, deserialized)
	}
}

func TestGitCommit_Serialize(t *testing.T) {
	tests := []struct {
		name   string
		commit *GitCommit
		want   string
	}{
		{
			name: "basic commit",
			commit: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Initial commit",
			},
			want: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nInitial commit",
		},
		{
			name: "merge commit",
			commit: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"parent":    {"1234567890abcdef1234567890abcdef12345678", "abcdef1234567890abcdef1234567890abcdef12"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Merge branch 'feature'",
			},
			want: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nparent 1234567890abcdef1234567890abcdef12345678\nparent abcdef1234567890abcdef1234567890abcdef12\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nMerge branch 'feature'",
		},
		{
			name: "commit with empty message",
			commit: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "",
			},
			want: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\n",
		},
		{
			name: "commit with encoding field",
			commit: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"encoding":  {"utf-8"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
				},
				Message: "Commit with encoding",
			},
			want: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nCommit with encoding",
		},
		{
			name: "commit with GPG signature",
			commit: &GitCommit{
				Kvlm: map[string][]string{
					"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
					"author":    {"John Doe <john@example.com> 1234567890 +0000"},
					"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
					"gpgsig":    {"-----BEGIN PGP SIGNATURE-----\niQIzBAABCAAdFiEEfake1234567890abcdefg=+0000FAKE\n=ABCD\n-----END PGP SIGNATURE-----"},
				},
				Message: "Signed commit",
			},
			want: "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\ngpgsig -----BEGIN PGP SIGNATURE-----\n iQIzBAABCAAdFiEEfake1234567890abcdefg=+0000FAKE\n =ABCD\n -----END PGP SIGNATURE-----\n\nSigned commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.commit.Serialize()
			if err != nil {
				t.Fatalf("Serialize() failed: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("Serialize() got:\n%s\n\nwant:\n%s", string(got), tt.want)
			}
		})
	}
}

func TestGitCommit_Serialize_MultiLineMessage(t *testing.T) {
	commit := &GitCommit{
		Kvlm: map[string][]string{
			"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
			"author":    {"John Doe <john@example.com> 1234567890 +0000"},
			"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
		},
		Message: "Add feature X\n\nThis commit adds the new feature X that allows users to:\n- Do something useful\n- Do something else\n\nAlso fixes bug Y.",
	}

	want := "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\n\nAdd feature X\n\nThis commit adds the new feature X that allows users to:\n- Do something useful\n- Do something else\n\nAlso fixes bug Y."

	got, err := commit.Serialize()
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	if string(got) != want {
		t.Errorf("Serialize() multi-line message failed:\ngot:\n%s\n\nwant:\n%s", string(got), want)
	}
}

func TestGitCommit_Serialize_Order(t *testing.T) {
	// Test that serialization respects the canonical order
	commit := &GitCommit{
		Kvlm: map[string][]string{
			"gpgsig":    {"fake signature"},
			"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
			"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
			"parent":    {"1234567890abcdef1234567890abcdef12345678"},
			"author":    {"John Doe <john@example.com> 1234567890 +0000"},
		},
		Message: "Test order",
	}

	got, err := commit.Serialize()
	if err != nil {
		t.Fatalf("Serialize() failed: %v", err)
	}

	// Should be in canonical order: tree, parent, author, committer, gpgsig
	want := "tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\nparent 1234567890abcdef1234567890abcdef12345678\nauthor John Doe <john@example.com> 1234567890 +0000\ncommitter Jane Doe <jane@example.com> 1234567890 +0000\ngpgsig fake signature\n\nTest order"

	if string(got) != want {
		t.Errorf("Serialize() order failed:\ngot:\n%s\n\nwant:\n%s", string(got), want)
	}
}

func TestGitCommit_Serialize_Consistency(t *testing.T) {
	commit := &GitCommit{
		Kvlm: map[string][]string{
			"tree":      {"4b825dc642cb6eb9a060e54bf8d69288fbee4904"},
			"author":    {"John Doe <john@example.com> 1234567890 +0000"},
			"committer": {"Jane Doe <jane@example.com> 1234567890 +0000"},
		},
		Message: "Consistency test",
	}

	// Call Serialize multiple times
	first, err := commit.Serialize()
	if err != nil {
		t.Fatalf("First Serialize() failed: %v", err)
	}

	second, err := commit.Serialize()
	if err != nil {
		t.Fatalf("Second Serialize() failed: %v", err)
	}

	third, err := commit.Serialize()
	if err != nil {
		t.Fatalf("Third Serialize() failed: %v", err)
	}

	// All should be identical
	if string(first) != string(second) {
		t.Errorf("Serialize() consistency failed: first and second calls differ")
	}

	if string(first) != string(third) {
		t.Errorf("Serialize() consistency failed: first and third calls differ")
	}

	if string(second) != string(third) {
		t.Errorf("Serialize() consistency failed: second and third calls differ")
	}
}
