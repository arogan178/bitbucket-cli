package output

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"
)

// ANSI colour codes. Kept small and deliberate so `bt` output is
// readable on every terminal theme (light and dark) and on non-ANSI
// sinks (we skip colour automatically when stdout isn't a TTY).
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
)

// ColorMode controls whether DiffRenderer emits ANSI escapes.
type ColorMode int

const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

// ParseColorMode accepts gh-style values.
func ParseColorMode(s string) ColorMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "always", "yes", "true", "1":
		return ColorAlways
	case "never", "no", "false", "0":
		return ColorNever
	default:
		return ColorAuto
	}
}

// RenderDiff reads a unified diff from `in` and writes it to `out`,
// applying ANSI colours to hunk headers, file banners, and ±lines when
// colour is enabled. It preserves every byte so diffs round-trip if
// the caller pipes to `patch` or `git apply` (just disable colour).
func RenderDiff(in io.Reader, out io.Writer, mode ColorMode) error {
	useColor := shouldColor(mode, out)
	br := bufio.NewReader(in)
	bw := bufio.NewWriter(out)

	for {
		line, err := br.ReadString('\n')
		if len(line) > 0 {
			if err := writeDiffLine(bw, line, useColor); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				return bw.Flush()
			}
			return err
		}
	}
}

func writeDiffLine(w *bufio.Writer, line string, useColor bool) error {
	if !useColor {
		return writeString(w, line)
	}
	switch {
	case strings.HasPrefix(line, "diff --git"),
		strings.HasPrefix(line, "diff "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "new file mode"),
		strings.HasPrefix(line, "deleted file mode"),
		strings.HasPrefix(line, "similarity index"),
		strings.HasPrefix(line, "rename from"),
		strings.HasPrefix(line, "rename to"),
		strings.HasPrefix(line, "Binary files"):
		return writeString(w, ansiBold+ansiBlue+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "--- "):
		return writeString(w, ansiBold+ansiRed+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "+++ "):
		return writeString(w, ansiBold+ansiGreen+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "@@"):
		return writeString(w, ansiCyan+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "+"):
		return writeString(w, ansiGreen+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "-"):
		return writeString(w, ansiRed+strings.TrimRight(line, "\n")+ansiReset+"\n")
	case strings.HasPrefix(line, "\\ No newline at end of file"):
		return writeString(w, ansiDim+strings.TrimRight(line, "\n")+ansiReset+"\n")
	default:
		return writeString(w, line)
	}
}

func writeString(w *bufio.Writer, s string) error {
	_, err := w.WriteString(s)
	return err
}

func shouldColor(mode ColorMode, w io.Writer) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	// Auto: enable when writing to a terminal and when NO_COLOR isn't set.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// OpenPager picks a pager for interactive output: honour $BT_PAGER,
// then $PAGER, then try `delta` (which renders diffs beautifully),
// then `less -R`, else fall back to stdout. The returned closer flushes
// and waits for the pager; if no pager is launched it's a no-op.
func OpenPager() (io.WriteCloser, func() error, error) {
	if !isTerminal(os.Stdout) {
		return nopWriteCloser{os.Stdout}, func() error { return nil }, nil
	}
	cmd := pagerCommand()
	if cmd == nil {
		return nopWriteCloser{os.Stdout}, func() error { return nil }, nil
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return nopWriteCloser{os.Stdout}, func() error { return nil }, err
	}
	if err := cmd.Start(); err != nil {
		return nopWriteCloser{os.Stdout}, func() error { return nil }, err
	}
	closer := func() error {
		_ = pipe.Close()
		return cmd.Wait()
	}
	return pipe, closer, nil
}

func pagerCommand() *exec.Cmd {
	if v := os.Getenv("BT_PAGER"); v != "" {
		return exec.Command("sh", "-c", v)
	}
	if v := os.Getenv("PAGER"); v != "" {
		return exec.Command("sh", "-c", v)
	}
	if path, err := exec.LookPath("delta"); err == nil {
		return exec.Command(path, "--paging=always")
	}
	if path, err := exec.LookPath("less"); err == nil {
		c := exec.Command(path, "-R", "-F", "-X")
		// -R: raw ANSI, -F: quit if one screen, -X: don't init alt screen.
		return c
	}
	return nil
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// RenderDiffStat scans a unified diff and prints a per-file summary
// (added/removed line counts) plus a total, similar to `git diff --stat`.
// Binary and rename-only changes are flagged rather than counted.
func RenderDiffStat(in io.Reader, out io.Writer, mode ColorMode) error {
	useColor := shouldColor(mode, out)
	type fileStat struct {
		path    string
		added   int
		removed int
		binary  bool
		rename  string
	}
	stats := map[string]*fileStat{}
	var current *fileStat

	br := bufio.NewScanner(in)
	br.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	for br.Scan() {
		line := br.Text()
		switch {
		case strings.HasPrefix(line, "diff --git"):
			// Parse `diff --git a/foo b/bar`; fall back to whole line.
			parts := strings.Fields(line)
			path := line
			if len(parts) >= 4 {
				path = strings.TrimPrefix(parts[3], "b/")
			}
			fs := &fileStat{path: path}
			stats[path] = fs
			current = fs
		case strings.HasPrefix(line, "Binary files"):
			if current != nil {
				current.binary = true
			}
		case strings.HasPrefix(line, "rename to "):
			if current != nil {
				current.rename = strings.TrimPrefix(line, "rename to ")
			}
		case strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- "):
			// headers; ignore for counts
		case strings.HasPrefix(line, "+"):
			if current != nil {
				current.added++
			}
		case strings.HasPrefix(line, "-"):
			if current != nil {
				current.removed++
			}
		}
	}
	if err := br.Err(); err != nil {
		return err
	}

	ordered := make([]*fileStat, 0, len(stats))
	for _, s := range stats {
		ordered = append(ordered, s)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].path < ordered[j].path })

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	var totAdd, totRm int
	for _, s := range ordered {
		totAdd += s.added
		totRm += s.removed
		label := s.path
		if s.rename != "" {
			label = s.path + " → " + s.rename
		}
		changes := fmt.Sprintf("%d", s.added+s.removed)
		if s.binary {
			changes = "Bin"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", label, changes, graph(s.added, s.removed, useColor)); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\n %d file(s) changed, %d insertion(s)(+), %d deletion(s)(-)\n", len(ordered), totAdd, totRm)
	return err
}

// graph returns a lipgloss-free +/- histogram (max 40 chars), optionally
// coloured.
func graph(add, rm int, color bool) string {
	total := add + rm
	if total == 0 {
		return ""
	}
	const width = 40
	var a, r int
	if total <= width {
		a, r = add, rm
	} else {
		a = add * width / total
		r = rm * width / total
		if a+r < width && add > 0 {
			a++
		}
	}
	pluses := strings.Repeat("+", a)
	minuses := strings.Repeat("-", r)
	if color {
		pluses = ansiGreen + pluses + ansiReset
		minuses = ansiRed + minuses + ansiReset
	}
	return pluses + minuses
}
