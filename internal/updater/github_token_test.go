package updater

import "testing"

// The doctor now tells a rate-limited user to set GITHUB_TOKEN, so the token
// has to actually be honoured — advice that does nothing is worse than none.
func TestGithubToken(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"nothing set", map[string]string{}, ""},
		{"GITHUB_TOKEN", map[string]string{"GITHUB_TOKEN": "ghp_a"}, "ghp_a"},
		{"GH_TOKEN, as the gh CLI sets it", map[string]string{"GH_TOKEN": "ghp_b"}, "ghp_b"},
		{"GITHUB_TOKEN wins", map[string]string{"GITHUB_TOKEN": "ghp_a", "GH_TOKEN": "ghp_b"}, "ghp_a"},
		{"blank is not a token", map[string]string{"GITHUB_TOKEN": "   "}, ""},
		{"surrounding space is trimmed", map[string]string{"GITHUB_TOKEN": " ghp_c\n"}, "ghp_c"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "")
			t.Setenv("GH_TOKEN", "")
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			if got := githubToken(); got != c.want {
				t.Errorf("githubToken() = %q, want %q", got, c.want)
			}
		})
	}
}
