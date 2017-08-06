package lint2hub

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	reDiffFileHeader = regexp.MustCompile(`^diff --git a/(.*) b/(.*)$`)
	reDiffHunk       = regexp.MustCompile(`^@@ -?(\d+)(?:,(\d+))? \+?(\d+)(?:,(\d)+)? @@$`)
	reDiffAddition   = regexp.MustCompile(`^\+`)
	reDiffContext    = regexp.MustCompile(`^ `)
)

type diff struct {
	filePositions map[string]map[int]int
}

func newDiff(diffStr string) *diff {
	filePositions := map[string]map[int]int{}
	for file, fileDiff := range splitDiffByFile(diffStr) {
		filePositions[file] = buildPositionMap(fileDiff)
	}

	return &diff{filePositions: filePositions}
}

// GetPosition retrieves the position of the file and lineNum in the diff.
// Returns (0, false) if the position is not present in the diff.
func (d *diff) GetPosition(file string, lineNum int) (position int, ok bool) {
	positions, ok := d.filePositions[file]
	if !ok {
		return 0, false
	}

	position, ok = positions[lineNum]
	return
}

// splitDiffByFile splits a large diff into smaller diffs per file
func splitDiffByFile(diffStr string) map[string]string {
	files := map[string]string{}
	file := ""
	firstHunk := -1
	position := 0

	scanner := &noallocLineScanner{str: diffStr}
	for {
		line, eof := scanner.NextLine()
		if eof {
			break
		}

		if matches := reDiffFileHeader.FindStringSubmatch(line); matches != nil {
			if file != "" && firstHunk > 0 {
				files[file] = diffStr[firstHunk:position]
			}

			file = matches[2]
			firstHunk = -1
		} else if firstHunk < 0 && reDiffHunk.MatchString(line) {
			firstHunk = position
		}

		position += len(line) + 1
	}
	if file != "" && firstHunk > 0 {
		files[file] = diffStr[firstHunk:position]
	}

	return files
}

// buildPositionMap builds a map of filename to a map of lineNum numbers in the new
// file to GitHub diff "positions". Positions are used to post comments on lines
// in the GitHub API.
//
// diff is a single file's diff, possibly extracted by splitDiffByFile.
func buildPositionMap(fileDiff string) map[int]int {
	positions := map[int]int{}
	lineNum := 0
	position := 0

	scanner := &noallocLineScanner{str: fileDiff}
	for {
		line, eof := scanner.NextLine()
		if eof {
			break
		}

		if matches := reDiffHunk.FindStringSubmatch(line); matches != nil {

			lineNum, _ = strconv.Atoi(matches[3])
		} else {
			if reDiffAddition.MatchString(line) {
				positions[lineNum] = position
				lineNum++
			} else if reDiffContext.MatchString(line) {
				lineNum++
			}
		}

		position++
	}

	return positions
}

type noallocLineScanner struct {
	str string
	pos int
}

func (s *noallocLineScanner) NextLine() (line string, eof bool) {
	if s.pos >= len(s.str) {
		return "", true
	}

	le := strings.Index(s.str[s.pos:], "\n")
	if le == -1 {
		le = len(s.str)
	}

	prevPos := s.pos
	s.pos += (le + 1)

	return s.str[prevPos : prevPos+le], false
}
