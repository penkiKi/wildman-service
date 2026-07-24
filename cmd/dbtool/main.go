package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"wildman-service/internal/infra/database"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: dbtool <backup|verify|restore>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	switch os.Args[1] {
	case "backup":
		flags := flag.NewFlagSet("backup", flag.ExitOnError)
		dataDir := flags.String("data-dir", "./data", "service data directory")
		output := flags.String("out", "", "new backup file")
		_ = flags.Parse(os.Args[2:])
		if *output == "" {
			fail("-out is required")
		}
		db, err := database.Open(ctx, *dataDir)
		if err != nil {
			fail(err.Error())
		}
		defer db.Close()
		if err := database.Backup(ctx, db, *output); err != nil {
			fail(err.Error())
		}
	case "verify":
		flags := flag.NewFlagSet("verify", flag.ExitOnError)
		file := flags.String("file", "", "backup file")
		_ = flags.Parse(os.Args[2:])
		if *file == "" {
			fail("-file is required")
		}
		result, err := database.Verify(ctx, *file)
		if err != nil {
			fail(err.Error())
		}
		_ = json.NewEncoder(os.Stdout).Encode(result)
	case "restore":
		flags := flag.NewFlagSet("restore", flag.ExitOnError)
		dataDir := flags.String("data-dir", "./data", "service data directory")
		file := flags.String("from", "", "verified backup file")
		confirm := flags.Bool("confirm", false, "confirm offline restore")
		_ = flags.Parse(os.Args[2:])
		if *file == "" || !*confirm {
			fail("-from and -confirm are required")
		}
		recoveryPath, err := database.Restore(ctx, *dataDir, *file)
		if err != nil {
			fail(err.Error())
		}
		if recoveryPath != "" {
			fmt.Println("previous database:", recoveryPath)
		}
	default:
		fail("unknown command")
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
