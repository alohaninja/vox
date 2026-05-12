//go:build !darwin

package inject

import (
	"errors"
	"testing"
)

func TestStubTypeTextEmptyReturnsNil(t *testing.T) {
	if err := TypeText(""); err != nil {
		t.Errorf("TypeText(\"\") should return nil, got %v", err)
	}
}

func TestStubTypeTextReturnsErrNotSupported(t *testing.T) {
	err := TypeText("hello")
	if err == nil {
		t.Fatal("TypeText(\"hello\") on stub must return an error")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("TypeText error should wrap ErrNotSupported, got: %v", err)
	}
}

func TestStubReadClipboardReturnsErrNotSupported(t *testing.T) {
	_, err := ReadClipboard()
	if err == nil {
		t.Fatal("ReadClipboard on stub must return an error")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ReadClipboard error should wrap ErrNotSupported, got: %v", err)
	}
}

func TestStubClearClipboardReturnsErrNotSupported(t *testing.T) {
	err := ClearClipboard()
	if err == nil {
		t.Fatal("ClearClipboard on stub must return an error")
	}
	if !errors.Is(err, ErrNotSupported) {
		t.Errorf("ClearClipboard error should wrap ErrNotSupported, got: %v", err)
	}
}
