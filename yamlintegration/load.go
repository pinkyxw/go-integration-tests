package yamlintegration

import (
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadTestCases recorre dirPath recursivamente y carga todos los archivos .yaml
// como TestCase. El orden de carga depende del orden de WalkDir (alfabético).
func LoadTestCases(dirPath string) ([]TestCase, error) {
	var cases []TestCase
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if d == nil || d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var tc TestCase
		if err := yaml.Unmarshal(b, &tc); err != nil {
			return err
		}
		cases = append(cases, tc)
		return nil
	})
	return cases, err
}
