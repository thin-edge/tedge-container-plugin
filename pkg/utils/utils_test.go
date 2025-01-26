package utils

import (
	"testing"
)

func Test_RootDir(t *testing.T) {
	test_cases := []struct {
		Path     string
		Expected string
	}{
		{
			Path:     "/data/example/foo.txt",
			Expected: "/data",
		},
		{
			Path:     "/data/example",
			Expected: "/data",
		},
		{
			Path:     "/test",
			Expected: "/test",
		},
		{
			Path:     "/",
			Expected: "/",
		},
		{
			Path:     "",
			Expected: "",
		},
	}
	for _, c := range test_cases {

		got := RootDir(c.Path)
		if got != c.Expected {
			t.Errorf("Invalid root dir. got=%v, wanted=%v", got, c.Expected)
		}

	}
}
