package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/client/cache"
)

type cacheTests struct {
	env e2e.TestEnv
}

// RunE2ETests is the main func to trigger the test suite
func RunE2ETests(env e2e.TestEnv) func(*testing.T) {
	c := &cacheTests{
		env: env,
	}

	return func(t *testing.T) {
		t.Run("cacheClean", c.testCacheClean)
	}
}

func (c *cacheTests) testCacheClean(t *testing.T) {
	tt := []struct {
		name    string
		options []string
		expect  string
		send    string
		output  string
		exit    int
	}{
		{
			name:   "clean cache normal",
			expect: "Do you want to continue? [N/y]",
			send:   "y",
			exit:   0,
		},
	}

	imagePath := filepath.Join(c.env.TestDir, "busybox.sif")
	for _, tc := range tt {
		c.env.RunSingularity(
			t,
			e2e.AsSubtest(tc.name),
			e2e.PreRun(func(t *testing.T) {
				h, err := cache.NewHandle(os.Getenv(cache.DirEnv))
				if err != nil {
					t.Fatalf("Could not create image cache handle: %v", err)
				}
				ensureDirEmpty(t, h.Library)

				e2e.PullImage(t, c.env, "library://library/default/busybox:1.26", imagePath)
				ensureDirNotEmpty(t, h.Library)
			}),
			e2e.WithCommand("cache clean"),
			e2e.WithArgs(tc.options...),
			e2e.ConsoleRun(
				e2e.ConsoleExpect(tc.expect),
				e2e.ConsoleSendLine(tc.send),
			),
			e2e.PostRun(func(t *testing.T) {
				h, err := cache.NewHandle(os.Getenv(cache.DirEnv))
				if err != nil {
					t.Fatalf("Could not create image cache handle: %v", err)
				}
				ensureDirEmpty(t, h.Library)
			}),
			e2e.ExpectExit(tc.exit),
		)
	}
}

func ensureDirEmpty(t *testing.T, dir string) {
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Could not read dir %q: %v", dir, err)
	}
	if len(fi) != 0 {
		t.Fatalf("Dir %q is not empty", dir)
	}
}

func ensureDirNotEmpty(t *testing.T, dir string) {
	fi, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Could not read dir %q: %v", dir, err)
	}
	if len(fi) == 0 {
		t.Fatalf("Dir %q is empty", dir)
	}
}
