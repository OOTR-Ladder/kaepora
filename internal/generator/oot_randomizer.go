package generator

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

type OOTRandomizer struct {
	version string
}

func NewOOTRandomizer(version string) *OOTRandomizer {
	return &OOTRandomizer{
		version: version,
	}
}

func (r *OOTRandomizer) Generate(settings, seed string) ([]byte, string, error) {
	outDir, err := ioutil.TempDir("", "oot-randomizer-output-")
	if err != nil {
		return nil, "", fmt.Errorf("unable to create output directory: %s", err)
	}
	defer os.RemoveAll(outDir)

	if err := r.run(outDir, settings, seed); err != nil {
		return nil, "", fmt.Errorf("unable to generate seed: %s", err)
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

func (r *OOTRandomizer) run(outDir, settings, seed string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	base := filepath.Join(wd, "resources/oot-randomizer")

	user, err := user.Current()
	if err != nil {
		return err
	}

	args := []string{
		"docker", "run", "--rm",
		"-u", fmt.Sprintf("%s:%s", user.Uid, user.Gid),
		"-v", base + "/ARCHIVE.bin:/opt/oot-randomizer/ARCHIVE.bin:ro",
		"-v", base + "/ZOOTDEC.z64:/opt/oot-randomizer/ZOOTDEC.z64:ro",
		"-v", filepath.Join(base, settings) + ":/opt/oot-randomizer/settings.json:ro",
		"-v", outDir + ":/opt/oot-randomizer/Output",
		"lp042/oot-randomizer:" + r.version,
		"--seed", seed,
		"--settings", "settings.json",
	}
	log.Printf("debug: %v", args)

	// There's no user input, unless the DB has been taken over.
	// nolint: gosec
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("stdout: %s", stdout.String())
		log.Printf("stderr: %s", stderr.String())
		return err
	}

	return nil
}
