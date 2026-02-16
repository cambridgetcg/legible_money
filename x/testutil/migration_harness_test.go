package testutil_test

import (
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/testutil"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
}

// TestMigrationHarnessBasicOps verifies the harness can seed, read, and iterate.
func TestMigrationHarnessBasicOps(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	// SetKV / GetKV
	suite.SetKV([]byte("key1"), []byte("value1"))
	got := suite.GetKV([]byte("key1"))
	if string(got) != "value1" {
		t.Fatalf("expected value1, got %s", string(got))
	}

	// HasKV
	if !suite.HasKV([]byte("key1")) {
		t.Fatal("expected key1 to exist")
	}
	if suite.HasKV([]byte("missing")) {
		t.Fatal("expected missing to not exist")
	}

	// DeleteKV
	suite.DeleteKV([]byte("key1"))
	if suite.HasKV([]byte("key1")) {
		t.Fatal("expected key1 to be deleted")
	}
}

// TestMigrationHarnessJSON verifies JSON round-trip helpers.
func TestMigrationHarnessJSON(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	type Record struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	// SetJSON
	suite.SetJSON([]byte("record1"), Record{Name: "alpha", Value: 42})

	// GetJSON
	var got Record
	suite.GetJSON([]byte("record1"), &got)
	if got.Name != "alpha" || got.Value != 42 {
		t.Fatalf("unexpected record: %+v", got)
	}
}

// TestMigrationHarnessIterate verifies prefix iteration.
func TestMigrationHarnessIterate(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	prefix := []byte("items/")
	suite.SetKV(append(prefix, []byte("a")...), []byte("1"))
	suite.SetKV(append(prefix, []byte("b")...), []byte("2"))
	suite.SetKV(append(prefix, []byte("c")...), []byte("3"))
	suite.SetKV([]byte("other/x"), []byte("excluded"))

	items := suite.IteratePrefix(prefix)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if string(items["items/a"]) != "1" {
		t.Fatalf("expected items/a=1, got %s", string(items["items/a"]))
	}
}

// TestMigrationHarnessSeedBatch verifies bulk seeding.
func TestMigrationHarnessSeedBatch(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	suite.SeedKVBatch(map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
		"c": []byte("3"),
	})

	if string(suite.GetKV([]byte("a"))) != "1" {
		t.Fatal("batch seed failed for key a")
	}
	if string(suite.GetKV([]byte("c"))) != "3" {
		t.Fatal("batch seed failed for key c")
	}
}

// TestMigrationHarnessSimulatedMigration demonstrates the full
// seed → migrate → verify pattern that real module migrations will use.
func TestMigrationHarnessSimulatedMigration(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	// v1 schema: {"name":"X"} without "active" field
	type V1Record struct {
		Name string `json:"name"`
	}
	// v2 schema: {"name":"X","active":true} with new field
	type V2Record struct {
		Name   string `json:"name"`
		Active bool   `json:"active"`
	}

	// Seed v1 data
	v1Data, _ := json.Marshal(V1Record{Name: "alice"})
	suite.SetKV([]byte("records/alice"), v1Data)
	v1Data2, _ := json.Marshal(V1Record{Name: "bob"})
	suite.SetKV([]byte("records/bob"), v1Data2)

	// Simulate migration: iterate, add default for new field
	records := suite.IteratePrefix([]byte("records/"))
	for key, val := range records {
		var v1 V1Record
		if err := json.Unmarshal(val, &v1); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		v2 := V2Record{Name: v1.Name, Active: true}
		v2Data, _ := json.Marshal(v2)
		suite.SetKV([]byte(key), v2Data)
	}

	// Verify v2 data
	var got V2Record
	suite.GetJSON([]byte("records/alice"), &got)
	if got.Name != "alice" || !got.Active {
		t.Fatalf("migration failed for alice: %+v", got)
	}
	suite.GetJSON([]byte("records/bob"), &got)
	if got.Name != "bob" || !got.Active {
		t.Fatalf("migration failed for bob: %+v", got)
	}
}

// TestMigrationHarnessAssertions verifies assertion helpers.
func TestMigrationHarnessAssertions(t *testing.T) {
	suite := testutil.NewMigrationTestSuite(t, "test_module")

	suite.SetKV([]byte("exists"), []byte("yes"))

	suite.AssertKeyExists([]byte("exists"))
	suite.AssertKeyNotExists([]byte("gone"))
}
