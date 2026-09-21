// Tests for parseGoTestOutput — the fresh-smoke validator.
// Reference: go-runtime/internal/cli/freshsmoke.go
package cli

import "testing"

// TestParseGoTestOutput covers the three contract scenarios:
//  1. Output containing both `ok` and `?` lines → validate passes
//  2. Output containing a `FAIL` line → validate fails
//  3. Output containing only `?` lines → validate passes
func TestParseGoTestOutput(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "ok_and_question_lines_pass",
			output: `ok  	github.com/ovav/ovav/internal/chronos	1.364s
ok  	github.com/ovav/ovav/internal/cli	1.095s
?   	github.com/ovav/ovav/cmd/session_greeting	[no test files]
?   	github.com/ovav/ovav/internal/email	[no test files]
ok  	github.com/ovav/ovav/internal/runtime	0.011s
FAIL`,
			want: true,
		},
		{
			name: "fail_line_fails",
			output: `ok  	github.com/ovav/ovav/internal/cli	1.095s
?   	github.com/ovav/ovav/cmd/ovav	[no test files]
FAIL	github.com/ovav/ovav/internal/ows	30.020s
FAIL`,
			want: false,
		},
		{
			name: "only_question_lines_pass",
			output: `?   	github.com/ovav/ovav/cmd/ovav-vault-tui	[no test files]
?   	github.com/ovav/ovav/cmd/plan	[no test files]
?   	github.com/ovav/ovav/cmd/session_greeting	[no test files]
?   	github.com/ovav/ovav/internal/connect	[no test files]
?   	github.com/ovav/ovav/internal/email	[no test files]`,
			want: true,
		},
		{
			name:   "empty_output_passes",
			output: "",
			want:   true,
		},
		{
			name: "trailing_fail_summary_without_package_passes",
			output: `ok  	github.com/ovav/ovav/internal/cli	1.095s
FAIL`,
			want: true,
		},
		{
			name: "fail_with_build_error_fails",
			output: `# github.com/ovav/ovav/internal/ows
./ows.go:10:2: undefined: foo
FAIL	github.com/ovav/ovav/internal/ows [build failed]
FAIL`,
			want: false,
		},
		{
			name: "spaces_separator_also_works",
			output: `ok github.com/ovav/ovav/internal/cli 1.095s
? github.com/ovav/ovav/internal/email [no test files]
FAIL github.com/ovav/ovav/internal/ows 30.020s`,
			want: false,
		},
		{
			name: "lone_FAIL_summary_without_package_is_ignored",
			output: `FAIL
`,
			want: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parseGoTestOutput(tc.output)
			if got != tc.want {
				t.Errorf("parseGoTestOutput() = %v, want %v", got, tc.want)
			}
		})
	}
}
