package converter

import (
	"os"
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"image"
	"image/jpeg"
	_ "image/png"
	"sync"

	// Used as underlying db driver
	_ "github.com/mattn/go-sqlite3"
)

// Initialize the output database with the required table schema.
func initializeDB(db *sql.DB) error {
	_, err := db.Exec("CREATE TABLE tiles (x INT, y INT, z INT, image BLOB, PRIMARY KEY (x, y, z))")
	if err != nil {
		return fmt.Errorf("error creating tiles table: %v", err)
	}

	_, err = db.Exec("CREATE INDEX IND on tiles (x,y,z)")
	if err != nil {
		return fmt.Errorf("error creating tiles table index: %v", err)
	}

	_, err = db.Exec("CREATE TABLE info (maxzoom INT, minzoom INT, tilenumbering TEXT)")
	if err != nil {
		return fmt.Errorf("error creating info table: %v", err)
	}

	return nil
}

// Convert raw tile data to JPEG with a specified quality.
func toJPEG(rawBytes []byte, quality int) ([]byte, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(rawBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to decode image config: %v", err)
	}
	slog.Debug(fmt.Sprintf("detected image format: %s", format))

	// Then call image.Decode if valid format
	if format != "jpeg" && format != "png" {
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	var jpegBuf bytes.Buffer
	err = jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}

	return jpegBuf.Bytes(), nil
}

// Process a single tile by converting to JPEG (if required) and calculating its coordinates.
func processTile(tileColumn, tileRow, zoomLevel int, tileData *[]byte, jpegQuality int) (int, int, []byte, error) {
	var jpgTileData []byte
	var err error

	if jpegQuality > 0 {
		jpgTileData, err = toJPEG(*tileData, jpegQuality)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("error converting to JPEG: %v", err)
		}
	} else {
		// If no conversion is required, just return the original data.
		jpgTileData = *tileData
	}

	y := (1 << uint(zoomLevel)) - 1 - tileRow
	return tileColumn, y, jpgTileData, nil
}

// Worker goroutine that processes tiles and inserts them into the database.
func worker(wg *sync.WaitGroup, tileChan <-chan []interface{}, stmt *sql.Stmt, jpegQuality int) {
	defer wg.Done()

	for tile := range tileChan {
		tileColumn := tile[0].(int)
		tileRow := tile[1].(int)
		zoomLevel := tile[2].(int)
		tileData := tile[3].([]byte)

		x, y, jpgTileData, err := processTile(tileColumn, tileRow, zoomLevel, &tileData, jpegQuality)
		if err != nil {
			fmt.Println(err)
			continue
		}

		_, err = stmt.Exec(x, y, 17-zoomLevel, jpgTileData)
		if err != nil {
			fmt.Println("error inserting tile:", err)
			continue
		}
	}
}

// Main function that orchestrates the conversion of MBTiles to OSM tiles.
func MbtilesToOsm(
	inputDBPath string,
	outputDBPath string,
	jpegQuality int,
	overwrite bool,
) error {
	slog.Info("Attempting conversion.", "input", inputDBPath, "output", outputDBPath)

	// Check if output file already exists
	if _, err := os.Stat(outputDBPath); err == nil {
		// File exists
		slog.Debug("File already exists", "filename", outputDBPath)
		if overwrite {
			// Overwrite is allowed, delete the existing file
			slog.Debug("Overwrite is allowed. Deleting existing file", "filename", outputDBPath)
			if err := os.Remove(outputDBPath); err != nil {
				return fmt.Errorf("failed to overwrite file (%v): %v", outputDBPath, err)
			}
		} else {
			// Overwrite is not allowed, return an error
			return fmt.Errorf("output file already exists: %v", outputDBPath)
		}
	} else if !os.IsNotExist(err) {
		// Some other error occurred (e.g., permission issues)
		return fmt.Errorf("failed to check file existence: %v", err)
	}

	// Check input file exists
	_, err := os.Stat(inputDBPath)
	if err != nil {
		return fmt.Errorf("file does not exist: %v", inputDBPath)
	}

	// Open input database
	inputDB, err := sql.Open("sqlite3", inputDBPath)
	if err != nil {
		return fmt.Errorf("error opening input database: %v", err)
	}
	defer inputDB.Close()

	// Open output database
	outputDB, err := sql.Open("sqlite3", outputDBPath)
	if err != nil {
		return fmt.Errorf("error opening output database: %v", err)
	}
	defer outputDB.Close()

	// Initialize output database schema
	err = initializeDB(outputDB)
	if err != nil {
		return err
	}

	// Prepare the insert statement for tiles
	stmt, err := outputDB.Prepare("INSERT INTO tiles (x, y, z, image) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing statement: %v", err)
	}
	defer stmt.Close()

	// Begin a transaction for better performance
	tx, err := outputDB.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	// Query all the tiles from the input database
	inputCursor, err := inputDB.Query("SELECT zoom_level, tile_column, tile_row, tile_data FROM tiles")
	if err != nil {
		return fmt.Errorf("error querying input database: %v", err)
	}
	defer inputCursor.Close()

	// Create a buffered channel for workers to process tiles concurrently
	tileChan := make(chan []interface{}, 100)
	var wg sync.WaitGroup

	// Launch multiple worker goroutines to process the tiles in parallel
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(&wg, tileChan, stmt, jpegQuality)
	}

	// Read from the input cursor and send tiles to the worker channel
	for inputCursor.Next() {
		var zoomLevel, tileColumn, tileRow int
		var tileData []byte
		err := inputCursor.Scan(&zoomLevel, &tileColumn, &tileRow, &tileData)
		if err != nil {
			fmt.Println("error scanning input row:", err)
			continue
		}

		tileChan <- []interface{}{tileColumn, tileRow, zoomLevel, tileData}
	}

	// Close the channel to signal no more tiles, and wait for workers to finish
	close(tileChan)
	wg.Wait()

	// Commit the transaction after all tiles have been processed
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	// Insert the zoom level info into the database
	_, err = outputDB.Exec(`
		INSERT INTO info
		(maxzoom, minzoom, tilenumbering)
		SELECT MAX(z), MIN(z), 'BigPlanet' FROM tiles
	`)
	if err != nil {
		return fmt.Errorf("error inserting into info table: %v", err)
	}

	slog.Info("Conversion completed successfully.")

	return nil
}
