package main

import (
	"os"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func toJPEG(rawBytes []byte, quality int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(rawBytes))
	if err != nil {
		return nil, err
	}

	// Convert the image to JPEG format with the specified quality
	var jpegBuf bytes.Buffer
	err = jpeg.Encode(&jpegBuf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, err
	}

	return jpegBuf.Bytes(), nil
}

func main() {
	input := flag.String("input", "", "input file")
	output := flag.String("output", "", "output file")
	jpegQuality := flag.Int("jpg", 0, "JPEG quality")
	force := flag.Bool("f", false, "force overwrite")
	flag.Parse()

	if *input == "" || *output == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if _, err := os.Stat(*output); err == nil {
		if *force {
			_ = os.Remove(*output)
		} else {
			fmt.Println("Output file already exists. Add -f option for overwrite")
			os.Exit(1)
		}
	}

	inputDB, err := sql.Open("sqlite3", *input)
	if err != nil {
		fmt.Println("Error opening input database:", err)
		os.Exit(1)
	}
	defer inputDB.Close()

	outputDB, err := sql.Open("sqlite3", *output)
	if err != nil {
		fmt.Println("Error opening output database:", err)
		os.Exit(1)
	}
	defer outputDB.Close()

	inputCursor, err := inputDB.Query("SELECT zoom_level, tile_column, tile_row, tile_data FROM tiles")
	if err != nil {
		fmt.Println("Error querying input database:", err)
		os.Exit(1)
	}
	defer inputCursor.Close()

	// Create tables in the output database
	_, err = outputDB.Exec("CREATE TABLE tiles (x INT, y INT, z INT, s INT, image BLOB, PRIMARY KEY (x, y, z, s))")
	if err != nil {
		fmt.Println("Error creating tiles table:", err)
		os.Exit(1)
	}

	_, err = outputDB.Exec("CREATE TABLE info (maxzoom INT, minzoom INT)")
	if err != nil {
		fmt.Println("Error creating info table:", err)
		os.Exit(1)
	}

	// Prepare the INSERT statement
	stmt, err := outputDB.Prepare("INSERT INTO tiles (x, y, z, s, image) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		fmt.Println("Error preparing statement:", err)
		os.Exit(1)
	}
	defer stmt.Close()

	tx, err := outputDB.Begin()
	if err != nil {
		fmt.Println("Error starting transaction:", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	var jpegBuf bytes.Buffer

	for inputCursor.Next() {
		var zoomLevel, tileColumn, tileRow int
		var tileData []byte
		err := inputCursor.Scan(&zoomLevel, &tileColumn, &tileRow, &tileData)
		if err != nil {
			fmt.Println("Error scanning input row:", err)
			continue
		}

		if *jpegQuality > 0 {
			// Convert tile data to JPEG format
			tileData, err = toJPEG(tileData, *jpegQuality)
			if err != nil {
				fmt.Println("Error converting to JPEG:", err)
				continue
			}
		}

		y := (1 << uint(zoomLevel)) - 1 - tileRow
		z := 17 - zoomLevel

		jpegBuf.Reset()
		_, err = stmt.Exec(tileColumn, y, z, 0, tileData)
		if err != nil {
			fmt.Println("Error inserting into tiles table:", err)
			continue
		}
	}

	_, err = tx.Exec("INSERT INTO info (maxzoom, minzoom) SELECT MAX(z), MIN(z) FROM tiles")
	if err != nil {
		fmt.Println("Error inserting into info table:", err)
		tx.Rollback()
		os.Exit(1)
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction:", err)
		os.Exit(1)
	}

	fmt.Println("Conversion completed successfully.")
}
