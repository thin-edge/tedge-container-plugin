package container

import "testing"

func Test_ResolveDockerIOImage(t *testing.T) {
	testcases := []struct {
		Image  string
		Expect string
	}{
		{
			Image:  "docker.io/library/app",
			Expect: "docker.io/library/app",
		},
		{
			Image:  "docker.io/other/app2",
			Expect: "docker.io/other/app2",
		},
		{
			Image:  "foo.io/other/app2",
			Expect: "foo.io/other/app2",
		},
	}

	for _, testcase := range testcases {
		got, _ := ResolveDockerIOImage(testcase.Image)

		if got != testcase.Expect {
			t.Errorf("Resolved image does not match expected. got=%s, wanted=%s", got, testcase.Expect)
		}
	}

}
