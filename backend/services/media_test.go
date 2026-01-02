package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFlattenFileValue(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want []string
	}{
		{"empty string", "", nil},
		{"single string", "file.jpg", []string{"file.jpg"}},
		{"string slice", []string{"a.png", "", "b.png"}, []string{"a.png", "b.png"}},
		{"interface slice", []interface{}{"a", 123, "b"}, []string{"a", "b"}},
		{"other type", 42, nil},
	}

	for _, tt := range tests {
		got := FlattenFileValue(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("%s: length mismatch got %v want %v", tt.name, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("%s: idx %d got %s want %s", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

func TestRemoveFileFromValue(t *testing.T) {
	tests := []struct {
		name     string
		in       interface{}
		filename string
		want     interface{}
		removed  bool
	}{
		{"remove string", "foo.jpg", "foo.jpg", "", true},
		{"no remove string", "foo.jpg", "bar.jpg", "foo.jpg", false},
		{"remove from slice", []string{"a", "b"}, "a", []string{"b"}, true},
		{"remove from iface slice", []interface{}{"a", "b"}, "b", []string{"a"}, true},
		{"no match slice", []string{"a"}, "x", []string{"a"}, false},
		{"other type", 123, "x", 123, false},
	}

	for _, tt := range tests {
		got, removed := RemoveFileFromValue(tt.in, tt.filename)
		if removed != tt.removed {
			t.Fatalf("%s: removed got %v want %v", tt.name, removed, tt.removed)
		}
		switch want := tt.want.(type) {
		case string:
			if got.(string) != want {
				t.Fatalf("%s: got %v want %v", tt.name, got, want)
			}
		case []string:
			gotSlice := got.([]string)
			if len(gotSlice) != len(want) {
				t.Fatalf("%s: len got %d want %d", tt.name, len(gotSlice), len(want))
			}
			for i := range want {
				if gotSlice[i] != want[i] {
					t.Fatalf("%s: idx %d got %s want %s", tt.name, i, gotSlice[i], want[i])
				}
			}
		default:
			if got != want {
				t.Fatalf("%s: got %v want %v", tt.name, got, want)
			}
		}
	}
}

func TestBuildMediaItem(t *testing.T) {
	tmp := t.TempDir()
	colID := "col1"
	recID := "rec1"
	filename := "test.txt"
	fullPath := filepath.Join(tmp, "storage", colID, recID)
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := []byte("hello world")
	if err := os.WriteFile(filepath.Join(fullPath, filename), content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	item, err := BuildMediaItem(tmp, "projects", colID, recID, "cover_image", filename, created)
	if err != nil {
		t.Fatalf("BuildMediaItem error: %v", err)
	}

	if item.Size != int64(len(content)) {
		t.Fatalf("size mismatch got %d want %d", item.Size, len(content))
	}
	if item.URL != "/api/files/col1/rec1/test.txt" {
		t.Fatalf("url mismatch: %s", item.URL)
	}
	if item.Collection != "projects" || item.Field != "cover_image" {
		t.Fatalf("collection/field mismatch: %+v", item)
	}
	if item.UploadedAt.IsZero() {
		t.Fatalf("uploaded_at should be set")
	}
	if item.Mime == "" {
		t.Fatalf("mime should not be empty")
	}
}
