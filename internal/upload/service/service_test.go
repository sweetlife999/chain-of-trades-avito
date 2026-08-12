package service

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveDetectsTypeByContentNotByName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		content       []byte
		wantExtension string
	}{
		{name: "png", content: encodePNG(t), wantExtension: ".png"},
		{name: "jpeg", content: encodeJPEG(t), wantExtension: ".jpg"},
		{name: "webp", content: webpHeader(), wantExtension: ".webp"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			directory := t.TempDir()
			service := newService(t, directory)

			url, err := service.Save(bytes.NewReader(testCase.content))
			if err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			name, found := strings.CutPrefix(url, URLPrefix)
			if !found || !strings.HasSuffix(name, testCase.wantExtension) {
				t.Fatalf("url = %q, want %s<uuid>%s", url, URLPrefix, testCase.wantExtension)
			}

			saved, err := os.ReadFile(filepath.Join(directory, name))
			if err != nil {
				t.Fatalf("read saved file: %v", err)
			}
			if !bytes.Equal(saved, testCase.content) {
				t.Fatalf("saved %d bytes, want %d", len(saved), len(testCase.content))
			}
		})
	}
}

// Расширение в имени файла клиента не должно ничего решать: под .jpg приезжает что угодно.
func TestSaveRejectsNonImage(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	service := newService(t, directory)

	_, err := service.Save(strings.NewReader("<?php echo 'hi'; ?>"))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("Save() error = %v, want %v", err, ErrValidation)
	}

	assertDirectoryEmpty(t, directory)
}

func TestSaveRejectsOversizedFileAndLeavesNothingBehind(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	service := newService(t, directory)

	// Начало настоящее, поэтому сниф пропускает файл дальше, и перебор ловится уже на записи.
	oversized := io.MultiReader(
		bytes.NewReader(encodePNG(t)),
		io.LimitReader(zeroReader{}, MaxFileBytes),
	)

	if _, err := service.Save(oversized); !errors.Is(err, ErrValidation) {
		t.Fatalf("Save() error = %v, want %v", err, ErrValidation)
	}

	assertDirectoryEmpty(t, directory)
}

func TestIsPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		value string
		want  bool
	}{
		{value: "/uploads/8db9f3e2.jpg", want: true},
		{value: "https://example.com/1.jpg", want: false},
		{value: "", want: false},
		{value: "/uploads/", want: false},
		// Протокол-относительный адрес: браузер сходил бы за картинкой на чужой домен.
		{value: "//evil.com/1.jpg", want: false},
		{value: "/uploads/../../etc/passwd", want: false},
		{value: "/uploads/..", want: false},
		{value: "/uploads/nested/1.jpg", want: false},
		{value: " /uploads/1.jpg", want: false},
	}

	for _, testCase := range cases {
		if got := IsPath(testCase.value); got != testCase.want {
			t.Errorf("IsPath(%q) = %v, want %v", testCase.value, got, testCase.want)
		}
	}
}

func newService(t *testing.T, directory string) *Service {
	t.Helper()

	service, err := New(directory)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return service
}

func assertDirectoryEmpty(t *testing.T, directory string) {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read uploads directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("после отказа в каталоге осталось %d файлов", len(entries))
	}
}

func encodePNG(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return buffer.Bytes()
}

func encodeJPEG(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	return buffer.Bytes()
}

// В stdlib нет кодировщика webp, а для снифа хватает заголовка контейнера.
func webpHeader() []byte {
	return append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 16)...)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	return len(p), nil
}
