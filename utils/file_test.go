package utils

import (
	"fmt"
	"testing"
)

func TestMatchAnyExt(t *testing.T) {
	var ext = []string{".png", ".jpg", ".jpeg"}
	for _, e := range ext {
		filepath := fmt.Sprintf("test_image%s", e)
		if !matchesAnyExt(filepath, ext) {
			t.Errorf("The extension %s did not match against %s", e, filepath)
		}
	}
	if matchesAnyExt("test_image.txt", ext) {
		t.Error("test_image.txt should fail to be matched")
	}
}
