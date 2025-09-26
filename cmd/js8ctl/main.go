package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dougsko/js8d/pkg/client"
)

var (
	socketPath = flag.String("socket", "/tmp/js8d.sock", "Unix socket path")
	command    = flag.String("cmd", "", "Command to send (e.g., 'STATUS', 'SEND:N0CALL Hello')")
)

func main() {
	flag.Parse()

	if *socketPath == "" {
		fmt.Fprintf(os.Stderr, "Socket path is required\n")
		os.Exit(1)
	}

	// If no command specified, show interactive help
	if *command == "" {
		if len(flag.Args()) > 0 {
			*command = strings.Join(flag.Args(), " ")
		} else {
			showHelp()
			return
		}
	}

	// Create socket client
	client := client.NewSocketClient(*socketPath)

	// Send command
	response, err := client.SendCommand(*command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Printf("%s\n", response.String())
}

func showHelp() {
	fmt.Println("js8ctl - JS8Call Daemon Control Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [options] <command>\n", os.Args[0])
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -socket <path>    Unix socket path (default: /tmp/js8d.sock)")
	fmt.Println("  -cmd <command>    Command to send")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  STATUS                    Get daemon status")
	fmt.Println("  MESSAGES                  Get recent messages")
	fmt.Println("  MESSAGES:10               Get last 10 messages")
	fmt.Println("  SEND:<to> <message>       Send a message")
	fmt.Println("  SEND:<message>            Send broadcast message")
	fmt.Println("  FREQUENCY:<freq>          Set radio frequency")
	fmt.Println("  RADIO                     Get radio status")
	fmt.Println("  PING                      Test connection")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s STATUS\n", os.Args[0])
	fmt.Printf("  %s 'SEND:N0CALL Hello from js8ctl'\n", os.Args[0])
	fmt.Printf("  %s MESSAGES:5\n", os.Args[0])
	fmt.Printf("  echo 'STATUS' | nc -U /tmp/js8d.sock\n")
}
