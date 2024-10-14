// Convert MBTile format --> OSMAnd SQLite format
// 1. Copy the .mbtiles to a new .sqlitedb file
// 2. Rename the columns to match the OSMAnd spec
// 3. Rewrite the zoom levels to match the OSMAnd spec
// 4. Remove the 'metadata' table and add an 'info' table

package converter

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"io"

	// We use a pure Go implementation of the SQLite driver
	// and not the mattn/go-sqlite3 C bindings.
	// Using the C bindings would require CGO_ENABLED=1, and
	// the SQLite C library to be bundled in the final binary.
	// While this is possible for Mac/Windows/Linux without
	// too much effort, it's not as easy for the Android build.
	_ "modernc.org/sqlite"
)

// Copy the input SQLite DB to the output path
func copyDB(inputDBPath, outputDBPath string, overwrite bool) error {
	// Check if output file already exists
	if _, err := os.Stat(outputDBPath); err == nil {
		slog.Debug("File already exists", "filename", outputDBPath)
		if overwrite {
			slog.Debug("Overwrite is allowed. Deleting existing file", "filename", outputDBPath)
			if err := os.Remove(outputDBPath); err != nil {
				return fmt.Errorf("failed to overwrite file (%v): %v", outputDBPath, err)
			}
		} else {
			return fmt.Errorf("output file already exists: %v", outputDBPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check file existence: %v", err)
	}

	// Open the input file for reading
	inputFile, err := os.Open(inputDBPath)
	if err != nil {
		return fmt.Errorf("error opening input database file: %v", err)
	}
	defer inputFile.Close()

	// Create the output file for writing
	outputFile, err := os.Create(outputDBPath)
	if err != nil {
		return fmt.Errorf("error creating output database file: %v", err)
	}
	defer outputFile.Close()

	// Copy the contents of the input DB to the output DB
	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("error copying database: %v", err)
	}

	return nil
}

// Update the schema and data of the output database
func updateDBSchema(db *sql.DB) error {
	// Rename columns
	_, err := db.Exec(`
		ALTER TABLE tiles RENAME COLUMN tile_column TO x;
		ALTER TABLE tiles RENAME COLUMN tile_row TO y;
		ALTER TABLE tiles RENAME COLUMN zoom_level TO z;
	`)
	if err != nil {
		return fmt.Errorf("error renaming columns: %v", err)
	}

	// Update the zoom levels (z column)
	_, err = db.Exec(`UPDATE tiles SET z = 17 - z`)
	if err != nil {
		return fmt.Errorf("error updating zoom levels: %v", err)
	}

	// Delete the metadata table
	_, err = db.Exec(`DROP TABLE IF EXISTS metadata`)
	if err != nil {
		return fmt.Errorf("error dropping metadata table: %v", err)
	}

	// Insert the new info table
	_, err = db.Exec(`
		CREATE TABLE info (maxzoom INT, minzoom INT, tilenumbering TEXT);
		INSERT INTO info (maxzoom, minzoom, tilenumbering) VALUES (17, 0, 'osm');
	`)
	if err != nil {
		return fmt.Errorf("error inserting info table: %v", err)
	}

	return nil
}

// Main function to convert MBTiles by copying and updating the DB schema and data
func MbtilesToOsm(
	inputDBPath string,
	outputDBPath string,
	overwrite bool,
) error {
	slog.Info("Starting conversion", "input", inputDBPath, "output", outputDBPath)

	// Copy the input database file to the output path
	err := copyDB(inputDBPath, outputDBPath, overwrite)
	if err != nil {
		return fmt.Errorf("error copying database: %v", err)
	}

	// Open the output database
	outputDB, err := sql.Open("sqlite", outputDBPath)
	outputDB.SetMaxOpenConns(1)
	if err != nil {
		return fmt.Errorf("error opening output database: %v", err)
	}
	defer outputDB.Close()

	// Update the schema and data in the copied DB
	err = updateDBSchema(outputDB)
	if err != nil {
		return fmt.Errorf("error updating database schema: %v", err)
	}

	slog.Info("Conversion completed successfully.", "output", outputDBPath)
	return nil
}
