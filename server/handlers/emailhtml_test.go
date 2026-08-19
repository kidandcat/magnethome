package handlers

import (
	"strings"
	"testing"
)

func TestWrapEmailHTML_fragmentGetsShell(t *testing.T) {
	out := wrapEmailHTML(`<table width="600"><tr><td>Hola</td></tr></table>`)
	if !strings.HasPrefix(out, "<!DOCTYPE html>") {
		t.Fatalf("expected document shell, got %q", out[:min(80, len(out))])
	}
	if !strings.Contains(out, `name="viewport"`) || !strings.Contains(out, `width=device-width`) {
		t.Fatal("missing viewport meta")
	}
	if !strings.Contains(out, "overflow-x:auto") || !strings.Contains(out, "min-width:0") {
		t.Fatal("missing overflow/shrink CSS for wide tables")
	}
	if !strings.Contains(out, `<table width="600"><tr><td>Hola</td></tr></table>`) {
		t.Fatal("original fragment should be preserved inside the shell")
	}
}

func TestWrapEmailHTML_injectsIntoExistingHead(t *testing.T) {
	in := `<!DOCTYPE html><html><HEAD><title>x</title></HEAD><body><p>hi</p></body></html>`
	out := wrapEmailHTML(in)
	head := out[strings.Index(strings.ToLower(out), "<head>") : strings.Index(strings.ToLower(out), "</head>")]
	if !strings.Contains(head, `name="viewport"`) {
		t.Fatal("viewport should be injected into existing head")
	}
	if !strings.Contains(out, "<p>hi</p>") {
		t.Fatal("body should be unchanged")
	}
	if strings.Count(strings.ToLower(out), "<html") != 1 {
		t.Fatal("should not wrap an existing document in another html")
	}
}

func TestWrapEmailHTML_htmlWithoutHead(t *testing.T) {
	out := wrapEmailHTML(`<html lang="es"><body><div style="width:800px">wide</div></body></html>`)
	if !strings.Contains(strings.ToLower(out), "<head>") {
		t.Fatal("expected a head to be inserted")
	}
	if !strings.Contains(out, `width=device-width`) {
		t.Fatal("missing viewport")
	}
	if !strings.Contains(out, `style="width:800px"`) {
		t.Fatal("sender markup should stay intact")
	}
}
