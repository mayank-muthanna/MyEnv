package schema

import "testing"

func TestParseCompilesRules(t *testing.T) {
	rules, err := Parse([]byte("PORT:\n  type: int\n  range: { min: 1, max: 10 }\nTOKEN:\n  type: string\n  pattern: '^tok_'\n"))
	if err != nil {
		t.Fatal(err)
	}
	if rules["PORT"].Range == nil || *rules["PORT"].Range.Max != 10 {
		t.Fatal("range was not parsed")
	}
	if !rules["TOKEN"].Pattern.MatchString("tok_value") {
		t.Fatal("pattern was not compiled")
	}
}

func TestParseRejectsRangeOnString(t *testing.T) {
	if _, err := Parse([]byte("NAME:\n  type: string\n  range: { min: 1 }\n")); err == nil {
		t.Fatal("expected schema error")
	}
}

func TestParseDocumentReadsIgnoreMetadata(t *testing.T) {
	document, err := ParseDocument([]byte("ignoreCode:\n  - CONVEX_*\nignoreUnused:\n  - CONVEX_SELF_HOSTED_ADMIN_KEY\nignorePaths:\n  - .nuxt/\nignoreRules:\n  - dynamic-env-access\nPORT:\n  type: int\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Schema) != 1 || document.Schema["PORT"].Type != "int" {
		t.Fatalf("unexpected schema: %#v", document.Schema)
	}
	if len(document.IgnoreCode) != 1 || len(document.IgnoreUnused) != 1 || len(document.IgnorePaths) != 1 || len(document.IgnoreRules) != 1 {
		t.Fatalf("unexpected ignore metadata: %#v", document)
	}
}
