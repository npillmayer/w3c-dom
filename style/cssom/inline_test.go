package cssom

import "testing"

func TestInlineDeclarations(t *testing.T) {
	for _, tc := range []struct {
		input, key, want string
		important        bool
	}{
		{`content: "Hello; A:B"`, "content", `"Hello; A:B"`, false},
		{`background-image: url("https://Example/a;b:c")`, "background-image", `url("https://Example/a;b:c")`, false},
		{`content: "A\";B";color:red`, "content", `"A\";B"`, false},
		{`color:red;color:blue`, "color", "blue", false},
		{`color:red!important;color:blue`, "color", "red", true},
		{`color:red;color:blue ! IMPORTANT`, "color", "blue", true},
		{`color:red!important;color:blue!important`, "color", "blue", true},
		{`content:"!important"`, "content", `"!important"`, false},
		{`COLOR: red ! /* comment */ important`, "color", "red", true},
		{`broken; color:blue`, "color", "blue", false},
	} {
		t.Run(tc.input, func(t *testing.T) {
			r := newLocalPseudoRule(tc.input)
			if got := string(r.Value(tc.key)); got != tc.want || r.IsImportant(tc.key) != tc.important {
				t.Fatalf("got %q important=%v", got, r.IsImportant(tc.key))
			}
		})
	}
}
