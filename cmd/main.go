package main

import (
	"flag"
	"fmt"
	"os"
	"log/slog"

	"gitlab.com/spwoodcock/mb2osm/converter"
)

var (
	verbose   = flag.Bool("v", false, "Show debug logs")
	overwrite = flag.Bool("f", false, "Force overwrite")
)

func configureLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	level := slog.LevelInfo

	if logLevel == "DEBUG" {
		level = slog.LevelDebug
	}

	// Initialize and set the default logger
	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		},
		))

	slog.SetDefault(logger)
}

func handleFlagsFirst() {
	flag.Parse()

	// Ensure correct flag parsing, returning an error for unknown flags
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mb2osm [-flags] input.mbtiles output.sqlitedb\n")
		flag.PrintDefaults()
	}
	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}
}

func handleAllArgs() []string {
	handleFlagsFirst()

	// Get the remaining positional arguments (input and output)
	args := flag.Args()

	// Enforce exactly two positional arguments
	if len(args) != 2 {
		fmt.Printf("Invalid input!\n\n")
		flag.Usage()
		os.Exit(1)
	}

	return args
}

func main() {
	args := handleAllArgs()

	inputFile := args[0]
	outputFile := args[1]

	// User verbose flag
	if *verbose {
		os.Setenv("LOG_LEVEL", "DEBUG")
	}
	configureLogger()

	// Check for output file overwrite
	if _, err := os.Stat(outputFile); err == nil {
		if !*overwrite {
			fmt.Println("Output file already exists. Use -f option to force overwrite")
			os.Exit(1)
		}
	}

	// Run main MbtilesToOsm function
	err := converter.MbtilesToOsm(
		inputFile,
		outputFile,
		*overwrite,
	)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
