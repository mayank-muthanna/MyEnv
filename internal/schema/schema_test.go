package schema

import (
	"strings"
	"testing"
)

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

func TestRenderIncludesCommentedIgnoreTemplate(t *testing.T) {
	contents, err := Render(Schema{"PORT": {Key: "PORT", Type: "int"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$'", "# Add pattern: '...'", "# ignoreCode:", "# ignoreUnused:"} {
		if !strings.Contains(string(contents), expected) {
			t.Fatalf("missing generated template content %q: %s", expected, contents)
		}
	}
	if _, err := Parse(contents); err != nil {
		t.Fatalf("rendered schema must parse: %v", err)
	}
}

func TestRenderDocumentPreservesIgnorePolicy(t *testing.T) {
	contents, err := RenderDocument(Document{
		Schema:       Schema{"PORT": {Key: "PORT", Type: "int"}},
		IgnoreCode:   []string{"DEPLOYMENT_SECRET"},
		IgnoreUnused: []string{"DEPLOYMENT_ONLY_SETTING"},
		IgnorePaths:  []string{".nuxt/"},
		IgnoreRules:  []string{"dynamic-env-access"},
	})
	if err != nil {
		t.Fatalf("render document: %v", err)
	}
	document, err := ParseDocument(contents)
	if err != nil {
		t.Fatalf("parse rendered document: %v", err)
	}
	if len(document.IgnoreCode) != 1 || document.IgnoreCode[0] != "DEPLOYMENT_SECRET" || len(document.IgnorePaths) != 1 || document.IgnorePaths[0] != ".nuxt/" {
		t.Fatalf("ignore policy lost: %#v", document)
	}
}

func TestRenderDocumentKeepsEncryptedPayloadLast(t *testing.T) {
	document := Document{
		Schema: Schema{"PORT": {Key: "PORT", Type: "int"}},
		EncryptedEnv: &EncryptedEnv{
			Version: 1, Algorithm: "AES-256-GCM", Compression: "gzip", Nonce: "nonce", Ciphertext: "ciphertext",
		},
	}
	contents, err := RenderDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "\nencryptedEnv:\n") {
		t.Fatalf("expected payload at bottom: %s", contents)
	}
	parsed, err := ParseDocument(contents)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.EncryptedEnv == nil || parsed.EncryptedEnv.Ciphertext != "ciphertext" {
		t.Fatal("encrypted payload did not round trip through schema")
	}
}
