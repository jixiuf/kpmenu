package main

import (
	"fmt"
	"log"
	"os"

	"ser1.net/claptrap/v4"
	"ser1.net/kpmenu/kpmenulib"
)

// Version is the version of kpmenu
const Version = "1.5.0"

func main() {
	cc := kpmenulib.InitializeFlags(nil)
	if cc.Bool("version") {
		fmt.Println(Version)
		os.Exit(0)
	}
	if cc.Bool("help") {
		claptrap.Usage()
		os.Exit(0)
	}
	config := kpmenulib.NewConfiguration()
	if err := kpmenulib.LoadConfig(cc, config); err != nil {
		log.Fatalf("loading config: %s", err)
		os.Exit(1)
	}

	menu, err := kpmenulib.NewMenu(config)
	if err != nil {
		log.Fatalf("creating menu: %s", err)
		os.Exit(1)
	}
	menu.ReloadConfig = func() error {
		return kpmenulib.LoadConfig(cc, config)
	}

	// Start client
	if err = kpmenulib.StartClient(); err != nil {
		// Failed to comunicate with server - start server
		err = kpmenulib.StartServer(menu)

		if err != nil {
			log.Fatalf("starting server: %s", err)
			os.Exit(1)
		} else {
			log.Printf("waiting for goroutines to end")
			// Wait for any goroutine (clipboard)
			menu.WaitGroup.Wait()
		}
	}
}
