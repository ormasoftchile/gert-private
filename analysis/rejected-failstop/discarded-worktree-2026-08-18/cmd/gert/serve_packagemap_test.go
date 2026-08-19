package main

import (
	"context"
	"path/filepath"
	"testing"

	internalparser "github.com/ormasoftchile/gert/internal/parser"
	internaltool "github.com/ormasoftchile/gert/internal/tool"
	"github.com/ormasoftchile/gert/pkg/platform"
	"github.com/ormasoftchile/gert/pkg/schema"
	toolpkg "github.com/ormasoftchile/gert/pkg/tool"
)

func TestServePackageMapBindingOverridesPlannerAndRuntimeRegistries(t *testing.T) {
	workspace := makeWorkDir(t)
	realPackage := filepath.Join(workspace, "packages", "real")
	overridePackage := filepath.Join(workspace, "packages", "vscode")
	writeServePackage(t, realPackage, "real-binding")
	writeServePackage(t, overridePackage, "vscode-binding")

	writeFile(t, filepath.Join(workspace, ".gert", "config.yaml"), `apiVersion: config/v1
requires:
  - package: acme.incident-routing
    version: "^1.0.0"
    path: ./packages/real
`)
	writeFile(t, filepath.Join(workspace, "vscode.package-map.yaml"), `apiVersion: config/v1
requires:
  - package: acme.incident-routing
    version: "^1.0.0"
    path: ./packages/vscode
`)
	runbookPath := filepath.Join(workspace, "route.runbook.yaml")
	writeFile(t, runbookPath, `apiVersion: runbook/v1
id: serve-package-map
name: serve package-map binding test
requires:
  - package: acme.incident-routing
    version: "^1.0.0"
toolRefs:
  - name: route
    package: acme.incident-routing
flow:
  - step:
      id: route
      type: tool
      tool:
        name: route
        action: select
  - step:
      id: done
      type: end
      outcome: { category: success, code: done }
`)

	parserImpl, err := internalparser.New(platform.Real())
	if err != nil {
		t.Fatalf("new parser: %v", err)
	}
	runbook, err := parserImpl.Parse(context.Background(), runbookPath)
	if err != nil {
		t.Fatalf("parse runbook: %v", err)
	}

	assertBinding := func(t *testing.T, packageMapPath, want string) {
		t.Helper()
		plannerRegistry := &plannerToolRegistry{tools: make(map[string]*schema.ToolDef)}
		runtimeRegistry := internaltool.NewOverlayRegistry(internaltool.NewMapRegistry([]toolpkg.ToolDef{{
			Name: "route", Command: "scanned-binding",
		}}))
		binder, err := newServingPackageBinder(workspace, packageMapPath, plannerRegistry, runtimeRegistry)
		if err != nil {
			t.Fatalf("new serving package binder: %v", err)
		}
		if err := binder.bind(runbook); err != nil {
			t.Fatalf("bind package map %q: %v", packageMapPath, err)
		}

		runtimeTool, ok := runtimeRegistry.Lookup("route")
		if !ok {
			t.Fatal("runtime registry did not receive the catalog binding")
		}
		if runtimeTool.Command != want {
			t.Fatalf("runtime binding command = %q, want %q", runtimeTool.Command, want)
		}
		plannerTool, err := plannerRegistry.Lookup(context.Background(), "route", "select")
		if err != nil {
			t.Fatalf("planner registry did not receive the catalog binding: %v", err)
		}
		if plannerTool.Actions["select"].Description != want {
			t.Fatalf("planner binding description = %q, want %q", plannerTool.Actions["select"].Description, want)
		}
	}

	assertBinding(t, "", "real-binding")
	assertBinding(t, "vscode.package-map.yaml", "vscode-binding")
}

func writeServePackage(t *testing.T, root, command string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "gert-package.yaml"), `apiVersion: tool-package/v1
meta:
  name: acme.incident-routing
  version: "1.0.0"
exports:
  tools:
    - id: route
      path: tools/route.tool.yaml
`)
	writeFile(t, filepath.Join(root, "tools", "route.tool.yaml"), `apiVersion: tool/v1
meta:
  name: route
  version: "1.0.0"
transport:
  mode: native
  command: `+command+`
actions:
  - name: select
    description: `+command+`
`)
}
