package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dpshade/pocket-prompt/internal/cli"
	"github.com/dpshade/pocket-prompt/internal/server"
	"github.com/dpshade/pocket-prompt/internal/service"
	"github.com/dpshade/pocket-prompt/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "0.1.0"

// killExistingServers finds and kills any running pocket-prompt URL server processes
func killExistingServers() error {
	// Find processes running pocket-prompt with --url-server
	cmd := exec.Command("pgrep", "-f", "pocket-prompt.*--url-server")
	output, err := cmd.Output()
	if err != nil {
		// No processes found or pgrep failed
		return nil
	}

	// Parse PIDs and kill them
	pids := strings.Fields(string(output))
	currentPID := os.Getpid()
	
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		
		// Don't kill ourselves
		if pid == currentPID {
			continue
		}
		
		fmt.Printf("Killing existing server process (PID %d)...\n", pid)
		
		// Send SIGTERM first for graceful shutdown
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	
	// Give processes time to shut down
	time.Sleep(1 * time.Second)
	return nil
}

func printHelp() {
	fmt.Printf(`pocket-prompt - Terminal-based AI prompt management

USAGE:
    pocket-prompt [OPTIONS] [COMMAND]

OPTIONS:
    --help          Show this help information
    --version       Print version information  
    --init          Initialize a new prompt library
    --url-server    Start URL server for iOS Shortcuts integration
    --restart       Kill any running URL server instances and restart
    --port          Port for URL server (default: 8080)
    --sync-interval Git sync interval in minutes (default: 5, 0 to disable)
    --no-git-sync   Disable periodic git synchronization

COMMANDS:
    (no command)       Start interactive TUI mode
    list, ls           List all prompts
    search <query>     Search prompts
    get, show <id>     Show a specific prompt
    create, new <id>   Create a new prompt
    edit <id>          Edit an existing prompt
    delete, rm <id>    Delete a prompt
    copy <id>          Copy prompt to clipboard
    render <id>        Render prompt with variables
    templates          List templates
    template           Template management (create, edit, delete, show)
    tags               List all tags
    archive            Manage archived prompts
    search-saved       Manage saved searches
    boolean-search     Boolean search operations (create, edit, delete, list, run)
    export             Export prompts and templates
    import             Import prompts and templates
    git                Git synchronization commands
    help               Show CLI command help

EXAMPLES:
    pocket-prompt                                    # Start interactive mode
    pocket-prompt --init                             # Initialize new library
    pocket-prompt --url-server                       # Start URL server for iOS
    pocket-prompt --url-server --restart            # Kill existing servers and restart
    pocket-prompt --url-server --port 9000          # Start server on port 9000
    pocket-prompt --url-server --sync-interval 1    # Sync every 1 minute
    pocket-prompt --url-server --no-git-sync        # Disable git sync
    pocket-prompt list --format table               # List prompts in table format
    pocket-prompt search "machine learning"         # Search prompts
    pocket-prompt create my-prompt --title "Test"   # Create new prompt
    pocket-prompt render my-prompt --var name=John  # Render with variables
    pocket-prompt template create my-template        # Create template
    pocket-prompt boolean-search run "(ai OR ml)"   # Boolean search
    pocket-prompt export all --output backup.json   # Export everything
    pocket-prompt git setup <repo-url>              # Setup git sync
    pocket-prompt help <command>                     # Get detailed help

STORAGE:
    Default directory: ~/.pocket-prompt
    Override with: POCKET_PROMPT_DIR=<path>

For more information, visit: https://github.com/dpshade/pocket-prompt
`)
}

func main() {
	var showVersion bool
	var initLib bool
	var showHelp bool
	var urlServer bool
	var restartServer bool
	var port int
	var syncInterval int
	var noGitSync bool

	flag.BoolVar(&showVersion, "version", false, "Print version information")
	flag.BoolVar(&initLib, "init", false, "Initialize a new prompt library")
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&urlServer, "url-server", false, "Start URL server for iOS Shortcuts integration")
	flag.BoolVar(&restartServer, "restart", false, "Kill any running URL server instances and restart")
	flag.IntVar(&port, "port", 8080, "Port for URL server")
	flag.IntVar(&syncInterval, "sync-interval", 5, "Git sync interval in minutes (0 to disable)")
	flag.BoolVar(&noGitSync, "no-git-sync", false, "Disable periodic git synchronization")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("pocket-prompt version %s\n", version)
		os.Exit(0)
	}

	// Validate restart flag usage
	if restartServer && !urlServer {
		fmt.Printf("Error: --restart flag can only be used with --url-server\n")
		os.Exit(1)
	}

	// Initialize service with file storage
	svc, err := service.NewService()
	if err != nil {
		fmt.Println(err)
		return
	}

	if initLib {
		if err := svc.InitLibrary(); err != nil {
			fmt.Println("Error initializing library:", err)
			return
		}
		fmt.Println("Initialized Pocket Prompt library")
		return
	}

	if urlServer || restartServer {
		// Handle restart flag - kill existing servers first
		if restartServer {
			fmt.Printf("Restarting URL server...\n")
			if err := killExistingServers(); err != nil {
				fmt.Printf("Warning: Error killing existing servers: %v\n", err)
			}
		}
		
		fmt.Printf("Starting URL server for iOS Shortcuts integration...\n")
		urlSrv := server.NewURLServer(svc, port)
		
		// Configure git sync
		if noGitSync || syncInterval == 0 {
			urlSrv.SetGitSync(false)
		} else {
			urlSrv.SetGitSync(true)
			urlSrv.SetSyncInterval(time.Duration(syncInterval) * time.Minute)
		}
		
		if err := urlSrv.Start(); err != nil {
			fmt.Printf("Error starting URL server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if we have command line arguments for CLI mode
	args := flag.Args()
	if len(args) > 0 {
		// CLI mode - execute command and exit
		cliHandler := cli.NewCLI(svc)
		if err := cliHandler.ExecuteCommand(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// No arguments provided - start TUI mode
	// Initialize TUI
	model, err := ui.NewModel(svc)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Start TUI program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		return
	}
}
