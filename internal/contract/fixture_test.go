package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillRequestFixturesValidateAgainstContract(t *testing.T) {
	fixtures := loadFixtureArray(t, "tests/fixtures/skill-contract/skill-request-valid.json")
	for index, fixture := range fixtures {
		if err := ValidateSkillRequestFixture(fixture); err != nil {
			t.Fatalf("valid request fixture %d failed: %v", index, err)
		}
	}
}

func TestInvalidSkillRequestFixturesAreRejected(t *testing.T) {
	fixtures := loadNamedFixtureArray(t, "tests/fixtures/skill-contract/skill-request-invalid.json")
	for _, fixture := range fixtures {
		if err := ValidateSkillRequestFixture(fixture.Value); err == nil {
			t.Fatalf("invalid request fixture %s was accepted", fixture.Name)
		}
	}
}

func TestSkillResponseFixturesValidateAgainstContract(t *testing.T) {
	fixtures := loadFixtureArray(t, "tests/fixtures/skill-contract/skill-response-valid.json")
	for index, fixture := range fixtures {
		if err := ValidateSkillResponseFixture(fixture); err != nil {
			t.Fatalf("valid response fixture %d failed: %v", index, err)
		}
	}
}

func TestInvalidSkillResponseFixturesAreRejected(t *testing.T) {
	fixtures := loadNamedFixtureArray(t, "tests/fixtures/skill-contract/skill-response-invalid.json")
	for _, fixture := range fixtures {
		if err := ValidateSkillResponseFixture(fixture.Value); err == nil {
			t.Fatalf("invalid response fixture %s was accepted", fixture.Name)
		}
	}
}

type namedFixture struct {
	Name  string         `json:"name"`
	Value map[string]any `json:"value"`
}

func loadFixtureArray(t *testing.T, relativePath string) []map[string]any {
	t.Helper()
	data := readFixture(t, relativePath)
	var fixtures []map[string]any
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("decode %s: %v", relativePath, err)
	}
	return fixtures
}

func loadNamedFixtureArray(t *testing.T, relativePath string) []namedFixture {
	t.Helper()
	data := readFixture(t, relativePath)
	var fixtures []namedFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("decode %s: %v", relativePath, err)
	}
	return fixtures
}

func readFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	roots := []string{".", "..", "../..", "../../..", "../../../.."}
	for _, root := range roots {
		data, err := os.ReadFile(filepath.Join(root, relativePath))
		if err == nil {
			return data
		}
	}
	t.Fatalf("read %s: fixture not found under test ancestor roots", relativePath)
	return nil
}
