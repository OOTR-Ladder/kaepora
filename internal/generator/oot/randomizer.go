package oot

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

type Randomizer struct {
	version string
}

func NewRandomizer(version string) *Randomizer {
	return &Randomizer{
		version: version,
	}
}

func (g *Randomizer) Generate(settingsName, seed string) ([]byte, string, error) {
	outDir, err := ioutil.TempDir("", "oot-randomizer-output-")
	if err != nil {
		return nil, "", fmt.Errorf("unable to create output directory: %s", err)
	}
	defer os.RemoveAll(outDir)

	base, err := getBaseDir()
	if err != nil {
		return nil, "", err
	}
	settingsPath := filepath.Join(base, settingsName)

	zpf, spoilerLog, err := g.run(outDir, settingsPath, seed)
	if err != nil {
		return nil, "", fmt.Errorf("unable to generate seed: %s", err)
	}

	return zpf, spoilerLog, nil
}

func readFirstGlob(pattern string) ([]byte, error) {
	names, err := filepath.Glob(pattern)
	if err != nil || len(names) != 1 {
		return nil, fmt.Errorf("could not find file with glob `%s`: %w", pattern, err)
	}

	out, err := ioutil.ReadFile(names[0])
	if err != nil {
		return nil, fmt.Errorf("unable to read seed back: %w", err)
	}

	return out, nil
}

func getBaseDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(wd, "resources/oot-randomizer"), nil
}

func (g *Randomizer) run(outDir, settings, seed string) ([]byte, string, error) {
	base, err := getBaseDir()
	if err != nil {
		return nil, "", err
	}

	user, err := user.Current()
	if err != nil {
		return nil, "", err
	}

	// There's no user input, unless the DB has been taken over.
	// nolint: gosec
	cmd := exec.Command(
		"docker", "run", "--rm",
		"-u", fmt.Sprintf("%s:%s", user.Uid, user.Gid),
		"-v", base+"/ARCHIVE.bin:/opt/oot-randomizer/ARCHIVE.bin:ro",
		"-v", base+"/ZOOTDEC.z64:/opt/oot-randomizer/ZOOTDEC.z64:ro",
		"-v", settings+":/opt/oot-randomizer/settings.json:ro",
		"-v", outDir+":/opt/oot-randomizer/Output",
		"lp042/oot-randomizer:"+g.version,
		"--seed", seed,
		"--settings", "settings.json",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("stdout: %s", stdout.String())
		log.Printf("stderr: %s", stderr.String())
		return nil, "", err
	}

	zpf, err := readFirstGlob(filepath.Join(outDir, "*.zpf"))
	if err != nil {
		return nil, "", err
	}

	spoilerLog, err := readFirstGlob(filepath.Join(outDir, "*_Spoiler.json"))
	if err != nil {
		return nil, "", err
	}

	return zpf, string(spoilerLog), nil
}
