package web

import (
	"fmt"
	"io/ioutil"
	"kaepora/internal/generator/oot"
	"kaepora/internal/generator/oot/settings"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) devSettingsRelations(w http.ResponseWriter, r *http.Request) {
	if err := s.writeSettingsRelationSVG(w, r); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}
}

func (s *Server) writeSettingsRelationSVG(w http.ResponseWriter, r *http.Request) error {
	dot, err := getSettingsRelationsDOT()
	if err != nil {
		return err
	}

	tmp, err := ioutil.TempFile("", "*.dot")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.Write([]byte(dot))
	tmp.Close()
	if err != nil {
		return err
	}

	cmd := exec.Command("dot", "-Tsvg", tmp.Name()) // nolint:gosec
	cmd.Stdout = w
	w.Header().Set("Content-Type", "image/svg+xml")
	s.cache(w, r, 1*time.Hour)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func getSettingsRelationsDOT() (string, error) {
	dir, err := oot.GetBaseDir()
	if err != nil {
		return "", err
	}

	s, err := settings.Load(filepath.Join(dir, settings.DefaultName))
	if err != nil {
		return "", err
	}

	var dot strings.Builder
	dot.WriteString("digraph G {\n")
	dot.WriteString("\trankdir = LR;\n")
	dot.WriteString("\tnode [ width = 2 ];\n")
	dot.WriteString("\toverlap = false;\n")
	dot.WriteString("\tsplines = true;\n")

	for key, possible := range s {
		for _, value := range possible {
			// Only display values with relations
			if len(value.Implies) == 0 {
				continue
			}

			for impliedKey, impliedValue := range value.Implies {
				dot.WriteString("\t")
				fmt.Fprintf(
					&dot,
					`"%s" -> "%s" [label="%v -> %v"];`,
					key, impliedKey,
					value.Value, impliedValue,
				)
				dot.WriteString("\n")
			}
		}
	}
	dot.WriteString("}\n")

	return dot.String(), nil
}
