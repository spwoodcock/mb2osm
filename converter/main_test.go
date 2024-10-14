package converter

import (
	"os"
	"database/sql"
	"testing"
)

func TestInvalidInput(t *testing.T) {
	MbtilesToOsm(
		"./testdata/file_does_not_exist.mbtiles",
		"dummyfile.sqlitedb",
		true,
	)
}

func TestConvert(t *testing.T) {
	inputFile := "./testdata/new_kru_monrovia.mbtiles"

	// Create a temporary output file
	outputFile, err := os.CreateTemp("", "new_kru_monrovia.sqlitedb")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	// Clean up after test
	defer os.Remove(outputFile.Name())

	err = MbtilesToOsm(
		inputFile,
		outputFile.Name(),
		true,
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	// Check output file generated
	_, err = os.Stat(outputFile.Name())
	if err != nil {
		t.Fatalf("file does not exist: %v", outputFile.Name())
	}

	// Check if the output file contains expected tables
	db, err := sql.Open("sqlite", outputFile.Name())
	if err != nil {
		t.Fatalf("Failed to open output database: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='tiles'")
	if err != nil {
		t.Fatalf("Failed to query output database: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("Expected tiles table, but it was not found")
	}
}
