package lint2hub

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSplitDiffByFile(t *testing.T) {
	cases := []struct {
		diff     string
		expected map[string]string
	}{
		{
			diff: `diff --git a/README.md b/README.md
index 639f958..81590fa 100644
--- a/README.md
+++ b/README.md
@@ -1 +1,3 @@
-# test-repo
\ No newline at end of file
+# test-repo
+
+Hello World 1234
diff --git a/FOO.md b/BAR.md
index 639f958..81590fa 100644
--- a/FOO.md
+++ b/BAR.md
@@ -1 +1,3 @@
+# test-repo
+
+Hello World 1234
`,
			expected: map[string]string{
				"README.md": `@@ -1 +1,3 @@
-# test-repo
\ No newline at end of file
+# test-repo
+
+Hello World 1234
`,
				"BAR.md": `@@ -1 +1,3 @@
+# test-repo
+
+Hello World 1234
`,
			},
		},
	}

	for i, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("index %d", i), func(t *testing.T) {
			t.Parallel()

			files := splitDiffByFile(tc.diff)
			if !reflect.DeepEqual(tc.expected, files) {
				t.Errorf("expected %v, got %v", tc.expected, files)
			}
		})
	}
}

func TestBuildPositionMap(t *testing.T) {
	cases := []struct {
		diff     string
		expected map[int]int
	}{
		{
			diff: `@@ -1,3 +1,2 @@
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
`,
			expected: map[int]int{
				1:   2,
				101: 12,
				102: 13,
				103: 14,
			},
		},
	}

	for i, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("index %d", i), func(t *testing.T) {
			t.Parallel()

			positions := buildPositionMap(tc.diff)
			if !reflect.DeepEqual(tc.expected, positions) {
				t.Errorf("expected %v, got %v", tc.expected, positions)
			}
		})
	}
}
