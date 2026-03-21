
package multicoder

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"mc/pkg/ai"
	"mc/pkg/shared"
)

// promptTagRe matches <<<prompt>>> ... <<</prompt>>> blocks, including newlines.
var promptTagRe = regexp.MustCompile(`(?s)<<<prompt>>>(.*?)<<<\/prompt>>>`)

// promptMatch records a single prompt found in the file along with its byte
// positions so we can write responses back without overlapping each other.
type promptMatch struct {
	index      int    // ordinal position in the file (0-based)
	start      int    // byte offset of the opening < in <<<prompt>>>
	end        int    // byte offset one past the closing > in <<</prompt>>>
	userPrompt string // trimmed text between the tags
	indent     string // leading whitespace on the line where the tag starts
}

// promptResult is the async result for one prompt.
type promptResult struct {
	index    int
	response string
	err      error
}

// HandleWatch is the entry point for `mc watch [-r] "<pattern>"`.
func HandleWatch(pattern string, recursive bool) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔════════════════════════════════════════════════════════════════╗")
	cyan.Println("║                      MC WATCH STARTED                          ║")
	cyan.Println("╚════════════════════════════════════════════════════════════════╝\n")

	if err := shared.LoadEnvFile(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load .env file: %v\n", err)
	}

	yellow.Printf("→ Scanning for files matching: %s (recursive=%v)\n", pattern, recursive)
	files, err := GatherFiles([]string{pattern}, recursive)
	if err != nil {
		return fmt.Errorf("failed to gather files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching pattern: %s", pattern)
	}
	green.Printf("  ✓ Found %d file(s) to watch\n\n", len(files))
	for _, f := range files {
		yellow.Printf("    • %s\n", f)
	}
	fmt.Println()

	mi, err := ai.NewModelInterface("", "")
	if err != nil {
		return fmt.Errorf("failed to initialise AI: %v", err)
	}

	model, err := resolveModel()
	if err != nil {
		return err
	}
	green.Printf("→ Using model: %s\n\n", model)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	for _, f := range files {
		abs, err := filepath.Abs(f)
		if err != nil {
			continue
		}
		if err := watcher.Add(abs); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not watch %s: %v\n", abs, err)
		}
	}

	cyan.Println("→ Watching for saves with <<<prompt>>> ... <<</prompt>>> tags.")
	cyan.Println("  Press Ctrl+C to stop.\n")

	var (
		mu       sync.Mutex
		inFlight = make(map[string]bool)
	)

	debounce := func(path string, handler func(string)) {
		mu.Lock()
		if inFlight[path] {
			mu.Unlock()
			return
		}
		inFlight[path] = true
		mu.Unlock()

		go func() {
			// Give the editor time to finish flushing all bytes to disk.
			time.Sleep(150 * time.Millisecond)
			handler(path)
			mu.Lock()
			delete(inFlight, path)
			mu.Unlock()
		}()
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			path := event.Name
			if !matchesPattern(pattern, path) {
				continue
			}
			debounce(path, func(p string) {
				if err := processFile(p, mi, model); err != nil {
					red := color.New(color.FgRed)
					red.Printf("  ✗ Error processing %s: %v\n", p, err)
				}
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

// matchesPattern returns true when the base name of path matches the glob.
func matchesPattern(pattern, path string) bool {
	base := filepath.Base(path)
	matched, err := filepath.Match(pattern, base)
	if err != nil {
		return false
	}
	return matched
}

// lineIndent returns the leading whitespace characters of the line that
// contains the byte at position pos within content.
func lineIndent(content string, pos int) string {
	// Walk backward to find the start of the line.
	lineStart := pos
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}
	// Walk forward to find the first non-whitespace character.
	end := lineStart
	for end < len(content) && (content[end] == ' ' || content[end] == '\t') {
		end++
	}
	return content[lineStart:end]
}

// applyIndent prefixes every line of text after the first with indent.
// The first line is left untouched because it will be placed inline where the
// tag was, which already carries the correct indentation from the source file.
func applyIndent(text, indent string) string {
	if indent == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i := 1; i < len(lines); i++ {
		// Only add indent to non-empty lines so we do not pad blank lines.
		if lines[i] != "" {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// processFile reads path, finds all <<<prompt>>>…<<</prompt>>> blocks, sends
// each to the AI concurrently, and as each response arrives it is written back
// into the file at the correct position — compensating for any byte-length
// changes introduced by earlier responses so nothing overlaps.
func processFile(path string, mi *ai.ModelInterface, model string) error {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}
	content := string(data)

	// Collect every match with its byte positions.
	locs := promptTagRe.FindAllStringIndex(content, -1)
	if len(locs) == 0 {
		return nil
	}

	cyan.Printf("\n→ %d prompt tag(s) detected in: %s\n", len(locs), path)

	matches := make([]promptMatch, 0, len(locs))
	for i, loc := range locs {
		full := content[loc[0]:loc[1]]
		sub := promptTagRe.FindStringSubmatch(full)
		if len(sub) < 2 {
			continue
		}
		userPrompt := strings.TrimSpace(sub[1])
		if userPrompt == "" {
			yellow.Printf("  ⚠ Prompt %d is empty — skipping\n", i)
			continue
		}
		matches = append(matches, promptMatch{
			index:      i,
			start:      loc[0],
			end:        loc[1],
			userPrompt: userPrompt,
			indent:     lineIndent(content, loc[0]),
		})
	}

	if len(matches) == 0 {
		return nil
	}

	systemPrompt := GetSystemPrompt()

	// Launch one goroutine per prompt; results arrive on resultCh.
	resultCh := make(chan promptResult, len(matches))
	var wg sync.WaitGroup

	for _, m := range matches {
		wg.Add(1)
		m := m // capture
		yellow.Printf("  → Sending prompt %d (%d chars) to AI...\n", m.index, len(m.userPrompt))
		go func() {
			defer wg.Done()
			resp, err := mi.SendToAI(m.userPrompt, model, 0, 0.7, systemPrompt, nil)
			resultCh <- promptResult{index: m.index, response: resp, err: err}
		}()
	}

	// Close the channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// responses maps promptMatch.index → response text.
	responses := make(map[int]string, len(matches))
	var firstErr error

	for res := range resultCh {
		if res.err != nil {
			red.Printf("  ✗ Prompt %d failed: %v\n", res.index, res.err)
			if firstErr == nil {
				firstErr = res.err
			}
			continue
		}
		green.Printf("  ✓ Response for prompt %d received (%d chars)\n", res.index, len(res.response))
		responses[res.index] = res.response
	}

	if len(responses) == 0 {
		return firstErr
	}

	// Apply responses to the file content in order (low index → high index).
	// We track a running byte offset so that each substitution correctly
	// compensates for the length difference introduced by all previous ones.
	currentContent := content
	offset := 0

	for _, m := range matches {
		resp, ok := responses[m.index]
		if !ok {
			continue
		}

		// Re-indent the response to match the indentation level of the prompt tag.
		resp = applyIndent(resp, m.indent)

		adjStart := m.start + offset
		adjEnd := m.end + offset

		// Safety check — positions must still be in bounds.
		if adjStart < 0 || adjEnd > len(currentContent) || adjStart > adjEnd {
			red.Printf("  ✗ Prompt %d: position out of bounds after prior substitutions — skipping\n", m.index)
			continue
		}

		currentContent = currentContent[:adjStart] + resp + currentContent[adjEnd:]
		offset += len(resp) - (m.end - m.start)
	}

	if currentContent == content {
		return firstErr
	}

	if err := os.WriteFile(path, []byte(currentContent), 0644); err != nil {
		return fmt.Errorf("could not write updated file: %w", err)
	}

	green.Printf("  ✓ File updated: %s\n\n", path)
	return firstErr
}

// resolveModel reads AI_TOOLS_MODEL from the environment and falls back to a
// sensible default.
func resolveModel() (string, error) {
	model := os.Getenv("AI_TOOLS_MODEL")
	if model == "" {
		model = "anthropic/claude-sonnet-4"
	}
	return model, nil
}
