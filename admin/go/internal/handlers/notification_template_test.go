package handlers

import (
	"strings"
	"testing"
)

func TestRenderTemplateSubstitutesVariables(t *testing.T) {
	got, err := renderTemplate("Hello {{name}}, your balance is {{amount}} {{symbol}}", map[string]string{
		"name":   "Ada",
		"amount": "12.5",
		"symbol": "ETH",
	})
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if got != "Hello Ada, your balance is 12.5 ETH" {
		t.Fatalf("unexpected render: %q", got)
	}
}

func TestRenderTemplateWithoutPlaceholdersIsUnchanged(t *testing.T) {
	const body = "Scheduled maintenance tonight."
	got, err := renderTemplate(body, nil)
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if got != body {
		t.Fatalf("expected the body to be returned unchanged, got %q", got)
	}
}

func TestRenderTemplateRejectsMissingVariables(t *testing.T) {
	_, err := renderTemplate("Hello {{name}} from {{org}}", map[string]string{"name": "Ada"})
	if err == nil {
		t.Fatal("expected an error for a missing variable")
	}
	if !strings.Contains(err.Error(), "org") {
		t.Fatalf("error should name the missing variable, got %v", err)
	}
}

func TestRenderTemplateRejectsMalformedPlaceholders(t *testing.T) {
	for name, body := range map[string]string{
		"unclosed": "Hello {{name",
		"empty":    "Hello {{}}",
	} {
		if _, err := renderTemplate(body, map[string]string{"name": "Ada"}); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

func TestRenderTemplateToleratesRepeatedAndWhitespacedKeys(t *testing.T) {
	got, err := renderTemplate("{{ name }} and {{name}}", map[string]string{"name": "Ada"})
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if got != "Ada and Ada" {
		t.Fatalf("unexpected render: %q", got)
	}
}

// TestRenderTemplateDoesNotInterpretValues guards against a variable value being
// re-scanned as a placeholder, which would allow template injection.
func TestRenderTemplateDoesNotInterpretValues(t *testing.T) {
	got, err := renderTemplate("{{a}}", map[string]string{"a": "{{b}}", "b": "LEAKED"})
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	if got != "{{b}}" {
		t.Fatalf("value was re-expanded: %q", got)
	}
}
