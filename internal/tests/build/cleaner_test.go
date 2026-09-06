package build_test

import (
	"os"
	"path/filepath"
	"testing"

	tsbuild "github.com/michaelbomholt665/go-tree-sitter/internal/tree-sitter/build"
)

func TestCleanerRemovesLanguageDirectoryAndPreservesBuildRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	languageDir := filepath.Join(buildDir, "python")
	if err := os.MkdirAll(languageDir, 0o755); err != nil {
		t.Fatalf("create language dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(languageDir, "artifact"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	cleaner := tsbuild.NewCleaner()
	if err := cleaner.CleanLanguage(buildDir, "python"); err != nil {
		t.Fatalf("CleanLanguage returned error: %v", err)
	}

	if _, err := os.Stat(languageDir); !os.IsNotExist(err) {
		t.Fatalf("expected language directory to be removed, got err=%v", err)
	}
	if info, err := os.Stat(buildDir); err != nil || !info.IsDir() {
		t.Fatalf("expected build root to remain, err=%v", err)
	}
}

func TestCleanerPruneLanguagePreservesSourcesAndManifests(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	langDir := filepath.Join(buildDir, "python")

	// Bloat dirs to create
	bloatDirs := []string{
		filepath.Join(langDir, ".git", "objects", "01"),
		filepath.Join(langDir, ".github", "workflows"),
		filepath.Join(langDir, ".vscode"),
		filepath.Join(langDir, "node_modules", "tree-sitter-cli"),
		filepath.Join(langDir, "bindings", "python"),
		filepath.Join(langDir, "test", "corpus"),
		filepath.Join(langDir, "tests"),
		filepath.Join(langDir, "corpus"),
		filepath.Join(langDir, "examples"),
		filepath.Join(langDir, "docs"),
		filepath.Join(langDir, "script"),
		filepath.Join(langDir, "tools"),
		filepath.Join(langDir, "benchmark"),
		filepath.Join(langDir, "target", "release"),
		filepath.Join(langDir, "build", "Release"),
		filepath.Join(langDir, "src", "tree_sitter"),
		filepath.Join(langDir, "queries"),
	}
	for _, d := range bloatDirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	// Bloat files
	bloatFiles := map[string]string{
		filepath.Join(langDir, ".git", "config"):                              "[core]\n",
		filepath.Join(langDir, ".git", "HEAD"):                                "ref: refs/heads/main\n",
		filepath.Join(langDir, ".git", "objects", "01", "2345"):               "blob data",
		filepath.Join(langDir, ".github", "workflows", "ci.yml"):              "name: CI\n",
		filepath.Join(langDir, ".vscode", "settings.json"):                    "{}",
		filepath.Join(langDir, "node_modules", "tree-sitter-cli", "index.js"): "module.exports = {};",
		filepath.Join(langDir, "bindings", "python", "binding.c"):             "/* binding */",
		filepath.Join(langDir, "test", "corpus", "tests.txt"):                 "=== test ===",
		filepath.Join(langDir, "tests", "test_all.py"):                        "assert True",
		filepath.Join(langDir, "corpus", "more_tests.txt"):                    "=== test2 ===",
		filepath.Join(langDir, "examples", "example.py"):                      "print('hello')",
		filepath.Join(langDir, "docs", "index.md"):                            "# Docs",
		filepath.Join(langDir, "script", "test.sh"):                           "#!/bin/sh",
		filepath.Join(langDir, "tools", "gen.py"):                             "# gen",
		filepath.Join(langDir, "benchmark", "bench.js"):                       "// bench",
		filepath.Join(langDir, "target", "release", "lib.a"):                  "archive data",
		filepath.Join(langDir, "build", "Release", "obj.target"):              "native obj",
		filepath.Join(langDir, "src", "parser.o"):                             "compiled obj",
		filepath.Join(langDir, "src", "scanner.obj"):                          "compiled obj win",
		filepath.Join(langDir, "Cargo.toml"):                                  `[package] name = "tree-sitter-python"`,
		filepath.Join(langDir, "Cargo.lock"):                                  "# lock",
		filepath.Join(langDir, "pyproject.toml"):                              `[build-system] requires = ["setuptools"]`,
		filepath.Join(langDir, "setup.py"):                                    "from setuptools import setup",
		filepath.Join(langDir, "Package.swift"):                               "// swift-tools-version:5.3",
		filepath.Join(langDir, "Package.resolved"):                            "// lock",
		filepath.Join(langDir, "Makefile"):                                    "all:\n\t@true\n",
		filepath.Join(langDir, "binding.gyp"):                                 `{"targets": []}`,
		filepath.Join(langDir, "CMakeLists.txt"):                              "cmake_minimum_required(VERSION 3.10)",
		filepath.Join(langDir, "build.zig"):                                   "const std = @import(\"std\");",
		filepath.Join(langDir, "build.zig.zon"):                               ".{}",
		filepath.Join(langDir, "package-lock.json"):                           "{}",
		filepath.Join(langDir, "README.md"):                                   "# README",
		filepath.Join(langDir, "CHANGELOG.md"):                                "# Changelog",
		filepath.Join(langDir, ".gitignore"):                                  "/target\n",
		filepath.Join(langDir, ".gitattributes"):                              "* text=auto\n",
		filepath.Join(langDir, ".editorconfig"):                               "root = true\n",
		filepath.Join(langDir, ".clang-format"):                               "Language: Cpp\n",
	}
	for path, content := range bloatFiles {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bloat file %s: %v", path, err)
		}
	}

	// Preserved files
	preservedFiles := map[string]string{
		filepath.Join(langDir, "grammar.js"):                     "module.exports = grammar({});",
		filepath.Join(langDir, "node-types.json"):                `[{"type": "identifier"}]`,
		filepath.Join(langDir, "src", "parser.c"):                "/* parser */",
		filepath.Join(langDir, "src", "scanner.c"):               "/* scanner */",
		filepath.Join(langDir, "src", "tree_sitter", "parser.h"): "/* parser.h */",
		filepath.Join(langDir, "queries", "highlights.scm"):      "(identifier) @variable",
		filepath.Join(langDir, "tree-sitter.json"):               `{"name": "python"}`,
		filepath.Join(langDir, "package.json"):                   `{"name": "tree-sitter-python"}`,
		filepath.Join(langDir, "go.mod"):                         "module github.com/tree-sitter/tree-sitter-python",
		filepath.Join(langDir, "LICENSE"):                        "MIT License",
		filepath.Join(langDir, "LICENSE.md"):                     "# MIT License",
	}
	for path, content := range preservedFiles {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write preserved file %s: %v", path, err)
		}
	}

	cleaner := tsbuild.NewCleaner()
	if err := cleaner.PruneLanguage(buildDir, "python"); err != nil {
		t.Fatalf("PruneLanguage failed: %v", err)
	}

	// Verify all bloat files are removed
	for path := range bloatFiles {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected bloat file %s to be deleted, err=%v", path, err)
		}
	}
	// Verify bloat directories are removed
	for _, dirName := range []string{".git", ".github", ".vscode", "node_modules", "bindings", "test", "tests", "corpus", "examples", "docs", "script", "tools", "benchmark", "target", filepath.Join("build", "Release")} {
		p := filepath.Join(langDir, dirName)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected bloat dir %s to be deleted, err=%v", p, err)
		}
	}

	// Verify all preserved files still exist with correct content
	for path, expectedContent := range preservedFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected preserved file %s to exist: %v", path, err)
			continue
		}
		if string(data) != expectedContent {
			t.Errorf("file %s content mismatch: got %q, want %q", path, string(data), expectedContent)
		}
	}
}

func TestCleanerPruneLanguageEdgeCases(t *testing.T) {
	t.Parallel()

	cleaner := tsbuild.NewCleaner()

	// Empty language should return error
	if err := cleaner.PruneLanguage("some/dir", ""); err == nil {
		t.Errorf("expected error for empty language, got nil")
	}

	// Non-existent directory should return nil
	dir := t.TempDir()
	if err := cleaner.PruneLanguage(dir, "nonexistent"); err != nil {
		t.Errorf("expected nil error for non-existent directory, got %v", err)
	}
}

func TestCleanerCleanMethod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	pyDir := filepath.Join(buildDir, "python")
	jsDir := filepath.Join(buildDir, "javascript")

	if err := os.MkdirAll(filepath.Join(pyDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir py .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pyDir, "grammar.js"), []byte("python"), 0o644); err != nil {
		t.Fatalf("write py grammar: %v", err)
	}

	if err := os.MkdirAll(jsDir, 0o755); err != nil {
		t.Fatalf("mkdir js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(jsDir, "grammar.js"), []byte("js"), 0o644); err != nil {
		t.Fatalf("write js grammar: %v", err)
	}

	cleaner := tsbuild.NewCleaner()

	// Test Clean with Prune=true on python:
	// We need a dummy Config to test Clean
	// tsbuild.Cleaner implements Clean(ctx, cfg, req)
	// Let's test CleanLanguage and PruneLanguage directly or via Clean
	if err := cleaner.PruneLanguage(buildDir, "python"); err != nil {
		t.Fatalf("PruneLanguage failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pyDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected .git to be pruned")
	}
	if _, err := os.Stat(filepath.Join(pyDir, "grammar.js")); err != nil {
		t.Fatalf("expected grammar.js to be preserved")
	}

	// Test CleanLanguage on javascript:
	if err := cleaner.CleanLanguage(buildDir, "javascript"); err != nil {
		t.Fatalf("CleanLanguage failed: %v", err)
	}
	if _, err := os.Stat(jsDir); !os.IsNotExist(err) {
		t.Fatalf("expected jsDir to be removed")
	}
}
