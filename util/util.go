// Package util provides generic utility functions for gobatfish, mirroring
// pybatfish.util.
package util

import (
	"archive/zip"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/81ueman/gobatfish/exception"
)

// Max length of snapshot/question names. Not 255 to accommodate potential
// folders/extensions, etc.
const maxFilenameLen = 150

// Minimum timestamp supported by the ZIP format.
var minZipTimestamp = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// nameSpecialChars are characters that must be escaped in a name. They should
// stay in sync with SPECIAL_CHARS in CommonParser.java.
const nameSpecialChars = " \t,\\&()[]@" + "!#$%^;?<>={}"

// Dictable is implemented by datamodel elements that expose a dictionary
// representation, mirroring the Python `dict()` method used by BfJsonEncoder.
type Dictable interface {
	Dict() map[string]any
}

// HTMLer is implemented by objects that provide a custom HTML representation,
// mirroring the Python `_repr_html_()` convention.
type HTMLer interface {
	HTML() string
}

// StringPtr returns a pointer to s.
func StringPtr(s string) *string { return &s }

// IntPtr returns a pointer to i.
func IntPtr(i int) *int { return &i }

// BoolPtr returns a pointer to b.
func BoolPtr(b bool) *bool { return &b }

// GetUUID generates and returns a random RFC 4122 version 4 UUID string.
func GetUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should not fail; fall back to a time based value so the
		// caller always gets a unique-ish identifier rather than panicking.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ValidateName checks whether a given entity name is valid, returning an error
// describing the problem if it is not. entityType defaults to "snapshot" when
// empty.
func ValidateName(name, entityType string) error {
	if entityType == "" {
		entityType = "snapshot"
	}
	capitalized := strings.ToUpper(entityType[:1]) + entityType[1:]
	if strings.Contains(name, "/") {
		return fmt.Errorf("%s name cannot contain slashes ('/')", capitalized)
	}
	if len(name) > maxFilenameLen {
		return fmt.Errorf("%s names cannot be longer than %d characters", capitalized, maxFilenameLen)
	}
	if strings.ToLower(name) == "settings" {
		return fmt.Errorf("'%s' is a reserved word. Please rename the %s", name, entityType)
	}
	for _, r := range name {
		if !isValidNameChar(r) {
			return fmt.Errorf("%s is not a valid name for %s", name, entityType)
		}
	}
	return nil
}

func isValidNameChar(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	return r == '-' || r == '_'
}

// ValidateQuestionName checks whether a question name is valid, returning a
// *exception.QuestionValidationError if it is not.
func ValidateQuestionName(name string) error {
	if strings.Contains(name, "/") {
		return exception.NewQuestionValidationError("Question name cannot contain slashes ('/')")
	}
	if len(name) > maxFilenameLen {
		return exception.NewQuestionValidationError(
			fmt.Sprintf("Question name cannot be longer than %d characters", maxFilenameLen))
	}
	return nil
}

// EscapeName escapes the given name string with double quotes if needed.
//
// A name should be quoted if it begins with '"', '/', or a digit, or if it
// contains a special character.
func EscapeName(s string) string {
	if len(s) == 0 {
		return s
	}
	needsQuote := strings.HasPrefix(s, `"`) || strings.HasPrefix(s, "/")
	if !needsQuote && s[0] >= '0' && s[0] <= '9' {
		needsQuote = true
	}
	if !needsQuote {
		for _, c := range nameSpecialChars {
			if strings.IndexRune(s, c) > 0 {
				needsQuote = true
				break
			}
		}
	}
	if needsQuote {
		return `"` + s + `"`
	}
	return s
}

// EscapeHTML escapes HTML special characters in s, matching Python's
// html.escape (which uses &quot; and &#x27;).
func EscapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#x27;",
	)
	return r.Replace(s)
}

// GetHTML returns the HTML representation of element, falling back to the
// escaped string representation.
func GetHTML(element any) string {
	if h, ok := element.(HTMLer); ok {
		return h.HTML()
	}
	return EscapeHTML(fmt.Sprint(element))
}

// ConditionalStr returns a concatenation of prefix, object and suffix.
//
// It returns the empty string if obj is nil or an empty container/string,
// mirroring pybatfish.util.conditional_str.
func ConditionalStr(prefix string, obj any, suffix string) string {
	if obj == nil {
		return ""
	}
	if sizeOf(obj) <= 0 {
		return ""
	}
	return strings.Join([]string{prefix, fmt.Sprint(obj), suffix}, " ")
}

func sizeOf(v any) int {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
		return rv.Len()
	default:
		return 1
	}
}

// ZipDir zips a specified directory and writes it to w.
//
// It mirrors pybatfish.util.zip_dir: the parent of dirPath is used as the
// archive root, directories are stored uncompressed and the archive includes
// directory entries.
func ZipDir(dirPath string, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	relRoot := filepath.Dir(filepath.Clean(dirPath))

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		arcname, err := filepath.Rel(relRoot, path)
		if err != nil {
			return err
		}
		arcname = filepath.ToSlash(arcname)
		if arcname == "." {
			arcname = ""
		}

		if info.IsDir() {
			header := &zip.FileHeader{
				Name:   strings.TrimSuffix(arcname, "/") + "/",
				Method: zip.Store,
			}
			if !info.ModTime().Before(minZipTimestamp) {
				header.SetModTime(info.ModTime())
			}
			_, err = zw.CreateHeader(header)
			return err
		}

		header := &zip.FileHeader{
			Name:   arcname,
			Method: zip.Deflate,
		}
		if !info.ModTime().Before(minZipTimestamp) {
			header.SetModTime(info.ModTime())
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(writer, f)
		return err
	})
}
