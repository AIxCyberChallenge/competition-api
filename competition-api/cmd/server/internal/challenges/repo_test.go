package challenges

import (
	"context"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aixcyberchallenge/competition-api/competition-api/internal/logger"
)

var testRepo = "https://github.com/krosenberg-kududyn/very-important-repository.git"

// THIS IS NOT IN THE RIGHT PLACE
func TestGitRepositoryActions(t *testing.T) {
	repoPath := t.TempDir()

	err := exec.Command("git", "-C", repoPath, "clone", testRepo, ".").Run()
	require.NoError(t, err, "failed to clone test repo")

	t.Run("Checkout", func(t *testing.T) { testCheckout(t, repoPath) })
	t.Run("RefToCommit", func(t *testing.T) { testRefToCommit(t, repoPath) })
	t.Run("Strip", func(t *testing.T) { testStrip(t, repoPath) })
	t.Run("Package", func(t *testing.T) { testPackage(t, repoPath) })
	t.Run("Generate Diff", func(t *testing.T) { testGenerateDiff(t, repoPath) })
}

func testCheckout(t *testing.T, repoPath string) {
	t.Run("InvalidRef", func(t *testing.T) {
		err := checkoutRef(context.Background(), repoPath, "foobar")
		assert.Error(t, err, "did not error on unexpected ref")
	})

	t.Run("ValidBranch", func(t *testing.T) {
		assert.NoFileExists(t, path.Join(repoPath, "c"), "file c should not exist on main branch")
		err := checkoutRef(context.Background(), repoPath, "very-important-branch")
		require.NoError(t, err, "failed to checkout existing branch")
		assert.FileExists(t, path.Join(repoPath, "c"), "file c should exist on branch")
	})

	t.Run("ValidCommit", func(t *testing.T) {
		assert.FileExists(t, path.Join(repoPath, "b"), "file b should exist on branch")
		err := checkoutRef(
			context.Background(),
			repoPath,
			"20a3e70624b870e1666c53256f518e2e1544d41d",
		)
		require.NoError(t, err, "failed to checkout existing ref")
		assert.NoFileExists(t, path.Join(repoPath, "b"), "file b should not exist on ref")
		assert.FileExists(t, path.Join(repoPath, "a"), "file a should exist on ref")
	})

	err := resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")
}

func testRefToCommit(t *testing.T, repoPath string) {
	err := resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")

	t.Run("ExistingTag", func(t *testing.T) {
		commit, err := refToCommit(context.Background(), repoPath, "1")
		require.NoError(t, err, "failed to get commit for existing ref")

		assert.Equal(
			t,
			"5185e0b10d857d8cd4d54fff4c0d9bc2092e1048",
			commit,
			"commit hash do not match",
		)
	})

	t.Run("NotExistingTag", func(t *testing.T) {
		_, err := refToCommit(context.Background(), repoPath, "foobar")
		assert.Error(t, err, "ref does not exist")
	})

	t.Run("ExistingBranch", func(t *testing.T) {
		commit, err := refToCommit(context.Background(), repoPath, "very-important-branch")
		require.NoError(t, err, "failed to get commit for existing ref")

		assert.Equal(
			t,
			"5185e0b10d857d8cd4d54fff4c0d9bc2092e1048",
			commit,
			"commit hash do not match",
		)
	})

	t.Run("NotExistingBranch", func(t *testing.T) {
		_, err := refToCommit(context.Background(), repoPath, "foobar")
		assert.Error(t, err, "ref does not exist")
	})
}

func testStrip(t *testing.T, repoPath string) {
	err := resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")

	t.Run("OneExistingFile", func(t *testing.T) {
		assert.FileExists(t, path.Join(repoPath, "a"), "file a should exist")
		err = stripRepo(context.Background(), []string{"a"}, repoPath)
		require.NoError(t, err, "failed to strip repo")
		assert.NoFileExists(t, path.Join(repoPath, "a"), "file a should not exist")
	})

	err = resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")

	t.Run("ManyExistingFile", func(t *testing.T) {
		assert.FileExists(t, path.Join(repoPath, "a"), "file a should exist")
		assert.FileExists(t, path.Join(repoPath, "b"), "file b should exist")
		err = stripRepo(context.Background(), []string{"a", "b"}, repoPath)
		require.NoError(t, err, "failed to strip repo")
		assert.NoFileExists(t, path.Join(repoPath, "a"), "file a should not exist")
		assert.NoFileExists(t, path.Join(repoPath, "b"), "file b should not exist")
	})

	err = resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")

	t.Run("NotExistingFile", func(t *testing.T) {
		assert.NoFileExists(t, path.Join(repoPath, "d"), "file d should not exist")
		err = stripRepo(context.Background(), []string{"d"}, repoPath)
		require.NoError(t, err, "failed to strip repo")
		assert.NoFileExists(t, path.Join(repoPath, "d"), "file d should not exist")
	})

	err = resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")

	t.Run("MixExistingFiles", func(t *testing.T) {
		assert.NoFileExists(t, path.Join(repoPath, "d"), "file d should not exist")
		assert.FileExists(t, path.Join(repoPath, "b"), "file b should exist")
		err = stripRepo(context.Background(), []string{"b", "d"}, repoPath)
		require.NoError(t, err, "failed to strip repo")
		assert.NoFileExists(t, path.Join(repoPath, "d"), "file d should not exist")
		assert.NoFileExists(t, path.Join(repoPath, "b"), "file b should not exist")
	})

	err = resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")
}

func testPackage(t *testing.T, repoPath string) {
	t.Run("PackageOnce", func(t *testing.T) {
		outDir := t.TempDir()

		tarFile, err := packageRepo(context.Background(), repoPath, outDir)
		require.NoError(t, err, "failed to package the repo")
		assert.FileExists(t, tarFile, "tar should exist after packaging")
	})

	t.Run("PackageSameRepoTwice", func(t *testing.T) {
		outDir := t.TempDir()

		tarFile1, err := packageRepo(context.Background(), repoPath, outDir)
		require.NoError(t, err, "failed to package the repo")
		assert.FileExists(t, tarFile1, "tar should exist after packaging")

		tarFile2, err := packageRepo(context.Background(), repoPath, outDir)
		require.NoError(t, err, "failed to package the repo")
		assert.FileExists(t, tarFile2, "tar should exist after packaging")

		entries, err := os.ReadDir(outDir)
		require.NoError(t, err, "failed to read contents of outDir")

		assert.Len(t, entries, 2, "there should exist exactly 2 files in out dir")
	})

	t.Run("PackageTwoDifferentRepos", func(t *testing.T) {
		outDir := t.TempDir()

		repo1 := t.TempDir()
		repo2 := t.TempDir()

		tarFile1, err := packageRepo(context.Background(), repo1, outDir)
		require.NoError(t, err, "failed to package the repo")
		assert.FileExists(t, tarFile1, "tar should exist after packaging")

		tarFile2, err := packageRepo(context.Background(), repo2, outDir)
		require.NoError(t, err, "failed to package the repo")
		assert.FileExists(t, tarFile2, "tar should exist after packaging")

		entries, err := os.ReadDir(outDir)
		require.NoError(t, err, "failed to read contents of outDir")
		assert.Len(t, entries, 2, "there should exist exactly 2 files in out dir")
	})
}

func testGenerateDiff(t *testing.T, repoPath string) {
	// test generate diff should be first used here to test that it can handle the branch not existing
	t.Run("NeverBeforeSeenRef", func(t *testing.T) {
		diffDir, err := generateDiff(
			context.TODO(),
			t.TempDir(),
			repoPath,
			"main",
			"test-generate-diff",
			"ref.diff",
			[]string{".aixcc", ".github", ".git", ".gitattributes"},
		)
		require.NoError(t, err, "failed to make diff")

		diffPath := filepath.Join(diffDir, "ref.diff")
		assert.FileExists(t, diffPath, "diff should exist")

		content, err := os.ReadFile(diffPath)
		require.NoError(t, err, "failed to read file")

		assert.Contains(t, string(content), "+a", "missing a addition")
		assert.Contains(t, string(content), "+b", "missing b addition")
	})

	err := resetRepo(context.Background(), repoPath)
	require.NoError(t, err, "failed to reset the repo")
}

func resetRepo(ctx context.Context, repoPath string) error {
	logger.Logger.InfoContext(ctx, "Resetting repo")
	err := exec.Command("git", "-C", repoPath, "reset", "--hard").Run()
	if err != nil {
		return err
	}

	err = checkoutRef(ctx, repoPath, "main")
	if err != nil {
		return err
	}

	return nil
}
