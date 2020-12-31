package registry

import "testing"

//nolint:goconst
func TestParseContainerImage(t *testing.T) {
	var imageName, reference = parseContainerImage("image:tag")
	if imageName != "image" && reference != "tag" {
		t.Errorf("parseContainerImage was incorrect, got: %s, %s want: %s, %s.", imageName, reference, "image", "tag")
	}

	imageName, reference = parseContainerImage("image")
	if imageName != "image" && reference != "latest" {
		t.Errorf("parseContainerImage was incorrect, got: %s, %s want: %s, %s.", imageName, reference, "image", "latest")
	}

	imageName, reference = parseContainerImage("image@abc")
	if imageName != "image" && reference != "abc" {
		t.Errorf("parseContainerImage was incorrect, got: %s, %s want: %s, %s.", imageName, reference, "image", "abc")
	}

	imageName, reference = parseContainerImage("image:tag@abc")
	if imageName != "image" && reference != "abc" {
		t.Errorf("parseContainerImage was incorrect, got: %s, %s want: %s, %s.", imageName, reference, "image", "abc")
	}
}
