package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// defaultMaxSourceLines is the maximum line limit for source files.
const defaultMaxSourceLines = 500

// defaultMaxTestLines is the maximum line limit for test files.
const defaultMaxTestLines = 800

var (
	testRegex       = regexp.MustCompile(`_test\.go$|\.spec\.ts$|\.test\.ts$|/test/|testutil/`)
	filepathRel     = filepath.Rel
	gitStatusRunner = gitStatusFiles
	gitStatusCmd    = exec.CommandContext
	scanFilesFn     = scanFiles
)

// DefaultPolicy returns a Policy populated with the project standard quality gates.
func DefaultPolicy() Policy {
	return Policy{
		MaxSourceLines:   defaultMaxSourceLines,
		MaxTestLines:     defaultMaxTestLines,
		EnforceTestFirst: false,
		SkipPatterns: []string{
			"node_modules/",
			"/dist/",
			"^dist/",
			"build/",
			"coverage/",
			".vscode-test/",
			"wailsjs/",
			`\.d\.ts$`,
			"vendor/",
			".git/",
		},
		FileExtensions: []string{
			".go",
			".ts",
			".vue",
		},
	}
}

// LoadPolicy reads and parses a JSON policy file, merging specified fields onto defaults.
func LoadPolicy(path string) (Policy, error) {
	pol := DefaultPolicy()
	if strings.TrimSpace(path) == "" {
		return pol, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return pol, fmt.Errorf("read policy file %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &pol); err != nil {
		return pol, fmt.Errorf("parse policy file %s: %w", path, err)
	}

	if pol.MaxSourceLines <= 0 {
		pol.MaxSourceLines = defaultMaxSourceLines
	}
	if pol.MaxTestLines <= 0 {
		pol.MaxTestLines = defaultMaxTestLines
	}
	if len(pol.SkipPatterns) == 0 {
		pol.SkipPatterns = DefaultPolicy().SkipPatterns
	}
	if len(pol.FileExtensions) == 0 {
		pol.FileExtensions = DefaultPolicy().FileExtensions
	}

	return pol, nil
}

// IsTestFile reports whether path designates a test or test-utility file.
func IsTestFile(path string) bool {
	normalized := filepath.ToSlash(path)
	return testRegex.MatchString(normalized)
}

// ShouldSkip reports whether path matches any of the given skip patterns.
func ShouldSkip(path string, skipPatterns []string) bool {
	normalized := filepath.ToSlash(path)
	for _, pattern := range skipPatterns {
		if strings.Contains(pattern, "^") || strings.Contains(pattern, "$") {
			if matched, _ := regexp.MatchString(pattern, normalized); matched {
				return true
			}
		} else if strings.Contains(normalized, pattern) {
			return true
		}
	}
	return false
}

// countFileLines counts the number of lines in a file.
func countFileLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := bufio.NewReader(f)
	count := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					count++
				}
				break
			}
			return count, err
		}
		count++
	}
	return count, nil
}

// scanFiles walks root collecting matching files adhering to extensions and skip patterns.
func scanFiles(root string, pol Policy) ([]string, error) {
	var collected []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepathRel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if ShouldSkip(rel+"/", pol.SkipPatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if ShouldSkip(rel, pol.SkipPatterns) {
			return nil
		}

		ext := filepath.Ext(path)
		for _, targetExt := range pol.FileExtensions {
			if ext == targetExt {
				collected = append(collected, path)
				break
			}
		}
		return nil
	})
	return collected, err
}

// CheckFileSizes validates file line limits against the specified policy.
func CheckFileSizes(ctx context.Context, root string, pol Policy, filePaths ...string) ([]Violation, error) {
	var targets []string
	if len(filePaths) == 0 {
		var err error
		targets, err = scanFilesFn(root, pol)
		if err != nil {
			return nil, fmt.Errorf("scan files in %s: %w", root, err)
		}
	} else {
		for _, fp := range filePaths {
			abs := fp
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(root, fp)
			}
			targets = append(targets, abs)
		}
	}

	var violations []Violation
	for _, target := range targets {
		if ctx.Err() != nil {
			return violations, ctx.Err()
		}

		rel, err := filepathRel(root, target)
		if err != nil {
			rel = target
		}
		rel = filepath.ToSlash(rel)

		if ShouldSkip(rel, pol.SkipPatterns) {
			continue
		}

		lines, err := countFileLines(target)
		if err != nil {
			continue
		}

		if IsTestFile(target) {
			if pol.MaxTestLines > 0 && lines > pol.MaxTestLines {
				violations = append(violations, Violation{
					File:      rel,
					Rule:      "max_test_lines",
					Details:   fmt.Sprintf("test file exceeds limit of %d lines (%d lines)", pol.MaxTestLines, lines),
					LineCount: lines,
					Limit:     pol.MaxTestLines,
				})
			}
		} else {
			if pol.MaxSourceLines > 0 && lines > pol.MaxSourceLines {
				violations = append(violations, Violation{
					File:      rel,
					Rule:      "max_source_lines",
					Details:   fmt.Sprintf("source file exceeds limit of %d lines (%d lines)", pol.MaxSourceLines, lines),
					LineCount: lines,
					Limit:     pol.MaxSourceLines,
				})
			}
		}
	}

	return violations, nil
}

// gitStatusFiles returns modified/added file paths from git status in the given directory.
func gitStatusFiles(ctx context.Context, root string) ([]string, error) {
	cmd := gitStatusCmd(ctx, "git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}
		entry := strings.TrimSpace(line[3:])
		if arrowIdx := strings.Index(entry, " -> "); arrowIdx != -1 {
			entry = entry[arrowIdx+4:]
		}
		files = append(files, entry)
	}
	return files, nil
}

// CheckTestFirst checks that whenever source changes are made, tests accompany them.
func CheckTestFirst(ctx context.Context, root string, files []string) ([]Violation, error) {
	var targets = files
	if len(targets) == 0 {
		var err error
		targets, err = gitStatusRunner(ctx, root)
		if err != nil {
			// Not a git repository or git error; ignore without failing.
			return nil, nil
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	var sourceFiles []string
	var hasTestModifications bool

	for _, f := range targets {
		if ShouldSkip(f, DefaultPolicy().SkipPatterns) {
			continue
		}
		if IsTestFile(f) {
			hasTestModifications = true
		} else {
			ext := filepath.Ext(f)
			if ext == ".go" || ext == ".ts" || ext == ".js" || ext == ".vue" {
				sourceFiles = append(sourceFiles, f)
			}
		}
	}

	if len(sourceFiles) == 0 || hasTestModifications {
		return nil, nil
	}

	// If source files were touched but no test files were touched, check whether tests exist for them.
	var violations []Violation
	for _, sf := range sourceFiles {
		ext := filepath.Ext(sf)
		base := strings.TrimSuffix(sf, ext)
		testCandidate := base + "_test" + ext
		if ext == ".ts" || ext == ".js" {
			testCandidate = base + ".test" + ext
		}

		fullTestPath := filepath.Join(root, testCandidate)
		if _, err := os.Stat(fullTestPath); os.IsNotExist(err) {
			violations = append(violations, Violation{
				File:    sf,
				Rule:    "test_first_required",
				Details: fmt.Sprintf("source file modified without corresponding test (%s missing)", testCandidate),
			})
		}
	}

	return violations, nil
}
