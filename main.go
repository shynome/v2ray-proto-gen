package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Masterminds/semver"
)

var v2Dir, grpcDir, baseDir string

func init() {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()
	if baseDir, err = os.Getwd(); err != nil {
		return
	}
	v2Dir = filepath.Join(baseDir, "v2ray-core")
	grpcDir = filepath.Join(baseDir, "v2ray-proto")
}

func mapVersions(tags []string) (versions []*semver.Version) {
	versions = make([]*semver.Version, len(tags))
	var err error
	for i, tag := range tags {
		versions[i], err = semver.NewVersion(tag)
		if err != nil {
			panic(err)
		}
	}
	// sort.Sort(semver.Collection(versions))
	return
}

func getTags(wd string, count uint) (tags []string, err error) {
	cmd := exec.Command("git", "for-each-ref", "refs/tags/", fmt.Sprintf("--count=%v", count), "--sort=-v:refname", `--format=%(refname:short)`)
	cmd.Dir = wd
	var ouput []byte
	if ouput, err = cmd.Output(); err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(ouput))
	for scanner.Scan() {
		tag := scanner.Text()
		tags = append(tags, tag)
	}
	return
}

func checkoutTag(wd string, tag string) (err error) {
	cmd := exec.Command("git", "checkout", tag)
	cmd.Dir = wd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func output2OS(cmd *exec.Cmd) {
	// return
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func syncProtoFile() (err error) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`tar -cC %v $(cd %v && find . | grep -E '\.proto$') | tar -xC %v`, v2Dir, v2Dir, grpcDir))
	output2OS(cmd)
	return cmd.Run()
}

func addGrpcFile2Git(tag string) (err error) {
	gitAdd := exec.Command("git", "add", "-A")
	gitAdd.Dir = grpcDir
	output2OS(gitAdd)
	if err = gitAdd.Run(); err != nil {
		return
	}
	gitCommit := exec.Command("git", "commit", "-m", fmt.Sprintf("sync v2ray version %v proto files", tag))
	gitCommit.Dir = grpcDir
	output2OS(gitCommit)
	err = gitCommit.Run()
	// ignore git commit error
	err = nil
	return
}

func commitGrpcTag(tag string) (err error) {
	if addGrpcFile2Git(tag); err != nil {
		return
	}
	cmd := exec.Command("git", "tag", tag)
	cmd.Dir = grpcDir
	output2OS(cmd)
	return cmd.Run()
}

func syncTag(tag string) (err error) {
	if err = checkoutTag(v2Dir, tag); err != nil {
		return
	}
	if err = syncProtoFile(); err != nil {
		return
	}
	if err = commitGrpcTag(tag); err != nil {
		return
	}
	return
}

func main() {
	var err error
	defer func() {
		if err != nil {
			panic(err)
		}
	}()

	var v2Tags, grpcTags []string
	if v2Tags, err = getTags(v2Dir, 20); err != nil {
		return
	}
	if grpcTags, err = getTags(grpcDir, 1); err != nil {
		return
	}
	v2Versions := mapVersions(v2Tags)
	grpcLatestTag := "v0.0.0"
	if len(grpcTags) != 0 {
		grpcLatestTag = grpcTags[0]
	}
	grpcLatestVersion, err := semver.NewVersion(grpcLatestTag)
	if err != nil {
		return
	}
	var versions = make([]*semver.Version, 0)
	for _, version := range v2Versions {
		if version.Compare(grpcLatestVersion) == 1 {
			versions = append(versions, version)
		}
	}

	for i := len(versions) - 1; i >= 0; i-- {
		version := versions[i]
		tag := version.Original()
		fmt.Printf("添加 version %v 的 proto 文件 \n", tag)
		if err = syncTag(tag); err != nil {
			return
		}
		// break
	}
}
