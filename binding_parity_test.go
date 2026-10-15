package main

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Wails turns every exported method on App into a JavaScript function, and the
// frontend describes those functions by hand in src/types/wails.ts. This guard
// fails when the two disagree: a method the frontend calls that Go does not
// have, a changed parameter count, or a changed result shape.

// goOnly lists exported App methods the frontend deliberately does not call
// through the binding (it uses the REST API for them).
var goOnly = map[string]string{
	"WatchWorkspace":        "workspace watching is driven by the daemon's own watcher",
	"UnwatchWorkspace":      "workspace watching is driven by the daemon's own watcher",
	"GetRecentActivities":   "the activity views read /api/activities over HTTP",
	"GetActivityTimeline":   "the activity views read /api/activities/timeline over HTTP",
	"GetProviderModels":     "the provider screens read models from the daemon's REST API",
	"RegisterCloudProvider": "superseded by the persisted provider-config CRUD (AddProviderConfig)",
	"RemoveProvider":        "superseded by the persisted provider-config CRUD (RemoveProviderConfig)",
}

type tsMethod struct {
	name   string
	params int
	result string // "void" or the Promise payload
}

func tsAppMethods(t *testing.T) map[string]tsMethod {
	t.Helper()
	raw, err := os.ReadFile("frontend/src/types/wails.ts")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	start := strings.Index(src, "App?: {")
	if start < 0 {
		t.Fatal("no App declaration in wails.ts")
	}
	body, _, _ := strings.Cut(src[start:], "\n        };")

	decl := regexp.MustCompile(`(?s)\n\s+(\w+)\((.*?)\): Promise<(.*?)>;`)
	out := map[string]tsMethod{}
	for _, m := range decl.FindAllStringSubmatch(body, -1) {
		params := 0
		if strings.TrimSpace(m[2]) != "" {
			// Each parameter is "name: Type"; commas inside generics do not occur here.
			params = strings.Count(m[2], ":")
		}
		out[m[1]] = tsMethod{name: m[1], params: params, result: m[3]}
	}
	if len(out) < 30 {
		t.Fatalf("parsed only %d bindings from wails.ts; the declaration format changed", len(out))
	}
	return out
}

func TestWailsBindingsMatchTheFrontendDeclaration(t *testing.T) {
	appType := reflect.TypeOf(&App{})
	errType := reflect.TypeOf((*error)(nil)).Elem()
	declared := tsAppMethods(t)

	for name, ts := range declared {
		m, ok := appType.MethodByName(name)
		if !ok {
			t.Errorf("wails.ts declares App.%s but the Go App has no such method", name)
			continue
		}
		if got := m.Type.NumIn() - 1; got != ts.params {
			t.Errorf("App.%s: Go takes %d parameters, wails.ts declares %d", name, got, ts.params)
		}
		outs := m.Type.NumOut()
		if outs > 0 && m.Type.Out(outs-1) == errType {
			outs--
		}
		wantVoid := ts.result == "void"
		if wantVoid && outs != 0 {
			t.Errorf("App.%s: wails.ts says Promise<void> but Go returns a value", name)
		}
		if !wantVoid && outs != 1 {
			t.Errorf("App.%s: wails.ts says Promise<%s> but Go returns %d values", name, ts.result, outs)
		}
	}

	for i := 0; i < appType.NumMethod(); i++ {
		name := appType.Method(i).Name
		if _, ok := declared[name]; ok {
			continue
		}
		if _, ok := goOnly[name]; !ok {
			t.Errorf("App.%s is exported to the frontend but neither declared in wails.ts nor listed in goOnly", name)
		}
	}
	for name := range goOnly {
		if _, ok := appType.MethodByName(name); !ok {
			t.Errorf("goOnly lists %s, which no longer exists", name)
		}
	}
}
