package config_test

import (
	"testing"
  "strings"
	"github.com/govau/torque/config"
)

func Test_LoadFile_NonExistentFile_ReturnsError(t *testing.T) {
	t.Run("Test non-existent file returns error", func(t *testing.T) {
		if err := config.LoadFile("NoSuchFile", &config.Settings{}); err == nil {
			t.Error("LoadFile() error: expected non existent file to return error")
		}
	})
}

func Test_Load_ValidYaml_ReturnsSettings(t *testing.T) {
  testYaml := `
  uaa_origin: test
  cfs:
    - api_href: https://api.example.com
      id: TEST
  orgs:
    - name: test-org
      spaces:
      - name: test-space
        skip_ids:
          - TEST 
        repos:
          - govau/test
  `
	t.Run("Test valid yaml returns settings", func(t *testing.T) {
    var settings config.Settings
    if err := config.Load(strings.NewReader(testYaml), &settings); err != nil {
			t.Errorf("Load() error: %v", err)
		} else {
      if settings.UaaOrigin != "test" {
        t.Errorf("Load() error: expected UaaOrigin to be test but was %s", settings.UaaOrigin)
      }
    }
	})
}

func Test_Load_BadCfId_ReturnsError(t *testing.T) {
  testYaml := `
  cfs:
    - id: TEST
  orgs:
    - spaces:
      - cf_ids:
          - BAD
  `
	t.Run("Test valid yaml returns settings", func(t *testing.T) {
    var settings config.Settings
    if err := config.Load(strings.NewReader(testYaml), &settings); err == nil {
			t.Errorf("Load() expected an error due to bad cf_ids")
		}
	})
}
