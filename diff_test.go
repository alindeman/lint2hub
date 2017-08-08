package lint2hub

import (
	"fmt"
	"testing"
)

func TestDiff(t *testing.T) {
	cases := []struct {
		diff      string
		positions map[string]map[int]int
	}{
		{
			diff: `diff --git a/README.md b/README.md
index abc1234..bcd3456 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,2 @@
-Howdy
+Hello
-Cruel
 World
@@ -50,53 +50,50 @@
 Goodbye
-One
-Two
-Three
@@ -100,100 +100,103 @@
 Hello
+One
+Two
+Three
diff --git a/foo/bar.rb b/foo/bar.rb
index abc1234..bcd3456 100644
--- a/foo/bar.rb
+++ b/foo/bar.rb
@@ -1,3 +1,3 @@ Wat
 Foo
-Bar
+Baz
Bob
`,
			positions: map[string]map[int]int{
				"README.md": {
					1:   2,
					2:   0, // not an addition line
					3:   0, // not present
					101: 12,
					102: 13,
					103: 14,
				},
				"foo/bar.rb": {
					2: 3,
				},
			},
		},
	}

	for i, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("index %d", i), func(t *testing.T) {
			t.Parallel()

			diff := newDiff(tc.diff)
			for file, positions := range tc.positions {
				for lineNum, position := range positions {
					actual, ok := diff.GetPosition(file, lineNum)
					if position == 0 && ok {
						t.Fatalf("lineNum %v: expected ok = false, but got %v", lineNum, ok)
					} else if position != 0 && !ok {
						t.Fatalf("lineNum %v: expected ok = true, but got %v", lineNum, ok)
					}

					if position != actual {
						t.Fatalf("lineNum %v: expected %v, got %v", lineNum, position, actual)
					}
				}
			}
		})
	}
}
