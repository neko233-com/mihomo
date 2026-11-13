package constant

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	assert.False(t, (&path{}).IsSafePath("/usr/share/metacubexd/"))
	assert.True(t, (&path{
		safePaths: []string{"/usr/share/metacubexd"},
	}).IsSafePath("/usr/share/metacubexd/"))

	assert.False(t, (&path{}).IsSafePath("../metacubexd/"))
	assert.True(t, (&path{
		homeDir:   "/usr/share/mihomo",
		safePaths: []string{"/usr/share/metacubexd"},
	}).IsSafePath("../metacubexd/"))
	assert.False(t, (&path{
		homeDir:   "/usr/share/mihomo",
		safePaths: []string{"/usr/share/ycad"},
	}).IsSafePath("../metacubexd/"))

	assert.False(t, (&path{}).IsSafePath("/opt/mykeys/key1.key"))
	assert.True(t, (&path{
		safePaths: []string{"/opt/mykeys"},
	}).IsSafePath("/opt/mykeys/key1.key"))
	assert.True(t, (&path{
		safePaths: []string{"/opt/mykeys/"},
	}).IsSafePath("/opt/mykeys/key1.key"))
	assert.True(t, (&path{
		safePaths: []string{"/opt/mykeys/key1.key"},
	}).IsSafePath("/opt/mykeys/key1.key"))

	assert.True(t, (&path{}).IsSafePath("key1.key"))
	assert.True(t, (&path{}).IsSafePath("./key1.key"))
	assert.True(t, (&path{}).IsSafePath("./mykey/key1.key"))
	assert.True(t, (&path{}).IsSafePath("./mykey/../key1.key"))
	assert.False(t, (&path{}).IsSafePath("./mykey/../../key1.key"))

}

func TestConstrainPathsDiscardsEnvironmentExceptions(t *testing.T) {
	old := Path
	defer func() { Path = old }()
	Path = &path{homeDir: t.TempDir(), allowUnsafePath: true, safePaths: []string{t.TempDir()}}
	ConstrainPaths()
	assert.False(t, Path.allowUnsafePath)
	assert.Empty(t, Path.safePaths)
	assert.False(t, Path.IsSafePath(filepath.Join(t.TempDir(), "outside")))
}
