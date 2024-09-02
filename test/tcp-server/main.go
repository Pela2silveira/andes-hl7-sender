// main.go
package main

import (
//	"bufio"
	"fmt"
	"net"
	"os"
)

const port = "5555"

func main() {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		os.Exit(1)
	}
	defer ln.Close()
	fmt.Printf("TCP server listening on port %s\n", port)

	for {
		// Accept a new connection
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Println("Client connected")

		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
    // Close the connection when we're done
    defer conn.Close()

    // Read incoming data
    buf := make([]byte, 1024)
    _, err := conn.Read(buf)
    if err != nil {
        fmt.Println(err)
        return
    }

    // Print the incoming data
    fmt.Printf("Received: %s", buf)
}
