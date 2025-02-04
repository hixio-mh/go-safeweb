// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package htmlinject

import (
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ExampleTransform() {
	const in = `
<html>
<head>
<link rel=preload as="script" src="gopher.js">
</head>
<body>
{{.Content}}
<script type="application/javascript">alert("script")</script>
<form>
First name:<br>
<input type="text" name="firstname"><br>
Last name:<br>
<input type="text" name="lastname">
</form>
</body>
</html>
`
	got, err := Transform(strings.NewReader(in), CSPNoncesDefault, XSRFTokensDefault)
	if err != nil {
		// handle error
		panic(err)
	}
	template.Must(template.New("example transform").Funcs(map[string]interface{}{
		"XSRFToken": func() string { return "XSRFToken-secret" },
		"CSPNonce":  func() string { return "CSPNonce-secret" },
	}).Parse(got)).Execute(os.Stdout, map[string]string{"Content": "This is some content"})
	// Output:
	// <html>
	// <head>
	// <link nonce="CSPNonce-secret" rel=preload as="script" src="gopher.js">
	// </head>
	// <body>
	// This is some content
	// <script nonce="CSPNonce-secret" type="application/javascript">alert("script")</script>
	// <form><input type="hidden" name="xsrf-token" value="XSRFToken-secret">
	// First name:<br>
	// <input type="text" name="firstname"><br>
	// Last name:<br>
	// <input type="text" name="lastname">
	// </form>
	// </body>
	// </html>
}

func BenchmarkTransform(b *testing.B) {
	b.ReportAllocs()
	var (
		config = []Config{CSPNoncesDefault, XSRFTokensDefault}
		in     = `
<html>
<head>
<link rel=preload as="script" src="gopher.js">
</head>
<body>
<script type="application/javascript">alert("script")</script>
<form>
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
</body>
</html>
`
		want = `
<html>
<head>
<link nonce="{{CSPNonce}}" rel=preload as="script" src="gopher.js">
</head>
<body>
<script nonce="{{CSPNonce}}" type="application/javascript">alert("script")</script>
<form><input type="hidden" name="xsrf-token" value="{{XSRFToken}}">
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
</body>
</html>
`
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := Transform(strings.NewReader(in), config...)
		if err != nil {
			b.Fatalf("Transform: got err %q, didn't want one", err)
		}
		if got != want {
			b.Errorf("got %q, want %q", got, want)
		}
	}
}
func TestTransform(t *testing.T) {
	var tests = []struct {
		name     string
		config   []Config
		in, want string
	}{
		{
			name: "nothing to change",
			in: `
<html>
<header><title>This is title</title></header>
<body>
Hello world
</body>
</html>
`,
			want: `
<html>
<header><title>This is title</title></header>
<body>
Hello world
</body>
</html>
`,
		},
		{
			name:   "add CSP nonces",
			config: []Config{CSPNoncesDefault},
			in: `
<html>
<head>
<link rel=preload as="script" src="gopher.js">
<style>
h1 {
  border: 5px solid yellow;
}
</style>
</head>
<body>
<script type="application/javascript">alert("script")</script>
</body>
</html>
`,
			want: `
<html>
<head>
<link nonce="{{CSPNonce}}" rel=preload as="script" src="gopher.js">
<style nonce="{{CSPNonce}}">
h1 {
  border: 5px solid yellow;
}
</style>
</head>
<body>
<script nonce="{{CSPNonce}}" type="application/javascript">alert("script")</script>
</body>
</html>
`,
		},
		{
			name:   "add XSRF protection",
			config: []Config{XSRFTokensDefault},
			in: `
<form>
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
`,
			want: `
<form><input type="hidden" name="xsrf-token" value="{{XSRFToken}}">
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
`,
		},
		{
			name:   "all configs",
			config: []Config{CSPNoncesDefault, XSRFTokensDefault},
			in: `
<html>
<head>
<link rel="stylesheet" href="styles.css">
<link rel=preload as="script" src="gopher.js">
</head>
<body>
<script type="application/javascript">alert("script")</script>
<form>
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
</body>
</html>
`,
			want: `
<html>
<head>
<link rel="stylesheet" href="styles.css">
<link nonce="{{CSPNonce}}" rel=preload as="script" src="gopher.js">
</head>
<body>
<script nonce="{{CSPNonce}}" type="application/javascript">alert("script")</script>
<form><input type="hidden" name="xsrf-token" value="{{XSRFToken}}">
  First name:<br>
  <input type="text" name="firstname"><br>
  Last name:<br>
  <input type="text" name="lastname">
</form>
</body>
</html>
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Transform(strings.NewReader(tt.in), tt.config...)
			if err != nil {
				t.Fatalf("Transform: got err %q, didn't want one", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("-want +got %s", diff)
			}
		})
	}
}
