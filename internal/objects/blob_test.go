package objects

import "testing"

func TestGitBlob_Type(t *testing.T) {
	blob := &GitBlob{}
	expected := "blob"
	if blob.Type() != expected {
		t.Errorf("Expected %q, got %q", expected, blob.Type())
	}
}

func TestGitBlob_Serialize(t *testing.T) {
	// Test basic serialization with non-empty data
	testData := []byte("Hello, World!")
	blob := &GitBlob{data: testData}

	serialized, err := blob.Serialize()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if string(serialized) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(serialized))
	}

	// Test serialization with empty data
	emptyBlob := &GitBlob{data: []byte{}}
	serializedEmpty, err := emptyBlob.Serialize()
	if err != nil {
		t.Errorf("Expected no error for empty data, got %v", err)
	}
	if len(serializedEmpty) != 0 {
		t.Errorf("Expected empty slice for empty data, got %v", serializedEmpty)
	}

	// Test serialization with larger data
	largeData := make([]byte, 1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	largeBlob := &GitBlob{data: largeData}
	serializedLarge, err := largeBlob.Serialize()
	if err != nil {
		t.Errorf("Expected no error for large data, got %v", err)
	}
	if len(serializedLarge) != len(largeData) {
		t.Errorf("Expected length %d, got %d", len(largeData), len(serializedLarge))
	}
	for i := range largeData {
		if serializedLarge[i] != largeData[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, largeData[i], serializedLarge[i])
			break
		}
	}

	// Test serialization with nil data
	nilBlob := &GitBlob{data: nil}
	serializedNil, err := nilBlob.Serialize()
	if err != nil {
		t.Errorf("Expected no error for nil data, got %v", err)
	}
	if serializedNil != nil {
		t.Errorf("Expected nil for nil data, got %v", serializedNil)
	}
}

func TestGitBlob_Deserialize(t *testing.T) {
	// Test basic deserialization with non-empty data
	testData := []byte("Hello, World!")
	blob := &GitBlob{}

	err := blob.Deserialize(testData)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if string(blob.data) != string(testData) {
		t.Errorf("Expected %q, got %q", string(testData), string(blob.data))
	}

	// Test deserialization with empty data
	emptyData := []byte{}
	emptyBlob := &GitBlob{}

	err = emptyBlob.Deserialize(emptyData)
	if err != nil {
		t.Errorf("Expected no error for empty data, got %v", err)
	}
	if len(emptyBlob.data) != 0 {
		t.Errorf("Expected empty slice for empty data, got %v", emptyBlob.data)
	}

	// Test deserialization with larger data
	largeData := make([]byte, 1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	largeBlob := &GitBlob{}

	err = largeBlob.Deserialize(largeData)
	if err != nil {
		t.Errorf("Expected no error for large data, got %v", err)
	}
	if len(largeBlob.data) != len(largeData) {
		t.Errorf("Expected length %d, got %d", len(largeData), len(largeBlob.data))
	}
	for i := range largeData {
		if largeBlob.data[i] != largeData[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, largeData[i], largeBlob.data[i])
			break
		}
	}

	// Test deserialization with nil data
	nilData := []byte(nil)
	nilBlob := &GitBlob{}

	err = nilBlob.Deserialize(nilData)
	if err != nil {
		t.Errorf("Expected no error for nil data, got %v", err)
	}
	if nilBlob.data != nil {
		t.Errorf("Expected nil for nil data, got %v", nilBlob.data)
	}

	// Test round-trip: Serialize then Deserialize
	originalData := []byte("Round trip test data")
	originalBlob := &GitBlob{data: originalData}

	serialized, err := originalBlob.Serialize()
	if err != nil {
		t.Errorf("Expected no error during serialization, got %v", err)
	}

	newBlob := &GitBlob{}
	err = newBlob.Deserialize(serialized)
	if err != nil {
		t.Errorf("Expected no error during deserialization, got %v", err)
	}

	if string(newBlob.data) != string(originalData) {
		t.Errorf("Round trip failed: expected %q, got %q", string(originalData), string(newBlob.data))
	}

	// Test multiple deserializations - verify data is overwritten
	firstData := []byte("First data")
	secondData := []byte("Second data")

	multiBlob := &GitBlob{}

	err = multiBlob.Deserialize(firstData)
	if err != nil {
		t.Errorf("Expected no error for first deserialization, got %v", err)
	}
	if string(multiBlob.data) != string(firstData) {
		t.Errorf("First deserialization failed: expected %q, got %q", string(firstData), string(multiBlob.data))
	}

	err = multiBlob.Deserialize(secondData)
	if err != nil {
		t.Errorf("Expected no error for second deserialization, got %v", err)
	}
	if string(multiBlob.data) != string(secondData) {
		t.Errorf("Second deserialization failed: expected %q, got %q", string(secondData), string(multiBlob.data))
	}
	if string(multiBlob.data) == string(firstData) {
		t.Errorf("Data was not overwritten: still contains first data %q", string(multiBlob.data))
	}
}
