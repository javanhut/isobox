package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/javanhut/isobox/internal/environment"
	"github.com/javanhut/isobox/pkg/ipkg"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		handleInit()
	case "enter":
		handleEnter()
	case "exec":
		handleExec()
	case "pkg":
		handlePackage()
	case "status":
		handleStatus()
	case "destroy":
		handleDestroy()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("IsoBox - Isolated Linux Development Environment")
	fmt.Println("\nUsage:")
	fmt.Println("  isobox init [path]       Initialize isolated environment in directory (default: current)")
	fmt.Println("  isobox enter             Enter the isolated environment shell")
	fmt.Println("  isobox exec <cmd>        Execute command in isolated environment")
	fmt.Println("  isobox status            Show environment status")
	fmt.Println("  isobox destroy           Remove isolated environment")
	fmt.Println("\nPackage Management:")
	fmt.Println("  isobox pkg install <pkg>  Install a package")
	fmt.Println("  isobox pkg remove <pkg>   Remove a package")
	fmt.Println("  isobox pkg list           List installed packages")
	fmt.Println("  isobox pkg update         Update package index")
}

func handleInit() {
	path := "."
	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	fmt.Printf("Initializing IsoBox environment in: %s\n", path)
	env, err := environment.Initialize(path)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Printf("\nIsoBox environment created successfully!\n")
	fmt.Printf("Location: %s\n", env.Root)
	fmt.Printf("\nTo enter the environment, run:\n")
	fmt.Printf("  cd %s && isobox enter\n", path)
}

func handleEnter() {
	env, err := environment.Load(".")
	if err != nil {
		log.Fatalf("Failed to load environment: %v\n\nRun 'isobox init' first to create an environment.", err)
	}

	if err := env.EnterShell(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("Failed to enter environment: %v", err)
	}
}

func handleExec() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: isobox exec <command> [args...]")
		os.Exit(1)
	}

	env, err := environment.Load(".")
	if err != nil {
		log.Fatalf("Failed to load environment: %v\n\nRun 'isobox init' first.", err)
	}

	cmd := os.Args[2:]
	if err := env.Execute(cmd); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

func handleStatus() {
	env, err := environment.Load(".")
	if err != nil {
		fmt.Println("No IsoBox environment found in current directory")
		os.Exit(1)
	}

	env.PrintStatus()
}

func handleDestroy() {
	env, err := environment.Load(".")
	if err != nil {
		fmt.Println("No IsoBox environment found in current directory")
		os.Exit(1)
	}

	fmt.Printf("Warning: This will destroy the IsoBox environment at: %s\n", env.Root)
	fmt.Print("Are you sure? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Cancelled")
		return
	}
	if err := env.Destroy(); err != nil {
		log.Fatalf("Failed to destroy environment: %v", err)
	}

	fmt.Println("Environment destroyed successfully")
}

func handlePackage() {
	env, err := environment.Load(".")
	if err != nil {
		log.Fatalf("No IsoBox environment found. Run 'isobox init' first.")
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: isobox pkg [install|remove|list|update] [args...]")
		os.Exit(1)
	}

	subcommand := os.Args[2]
	pm := ipkg.NewPackageManager(env.Root)

	switch subcommand {
	case "install":
		if len(os.Args) < 4 {
			fmt.Println("Usage: isobox pkg install <package>")
			os.Exit(1)
		}
		if err := pm.Install(os.Args[3]); err != nil {
			log.Fatalf("Failed to install package: %v", err)
		}
	case "remove":
		if len(os.Args) < 4 {
			fmt.Println("Usage: isobox pkg remove <package>")
			os.Exit(1)
		}
		if err := pm.Remove(os.Args[3]); err != nil {
			log.Fatalf("Failed to remove package: %v", err)
		}
	case "list":
		if err := pm.List(); err != nil {
			log.Fatalf("Failed to list packages: %v", err)
		}
	case "update":
		if err := pm.Update(); err != nil {
			log.Fatalf("Failed to update package index: %v", err)
		}
	default:
		fmt.Printf("Unknown pkg subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}
