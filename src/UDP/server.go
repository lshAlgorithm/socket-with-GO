package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
)

type UserSession struct {
	Name     string
	Password string
	Addr     net.Addr
	LoggedIn bool
}

var sessions = make(map[string]*UserSession)
var mutex = &sync.Mutex{}

func IsUserLoggedIn(username string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	if session, exists := sessions[username]; exists {
		return session.LoggedIn
	}
	return false
}

func handlePacket(conn *net.UDPConn, packet []byte, addr *net.UDPAddr) {
	scanner := bufio.NewScanner(bytes.NewBuffer(packet))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case line == "EXIT":
			// UDP doesn't have an explicit 'close' operation.
			return
		case strings.HasPrefix(line, "LOGIN"):
			parts := strings.Fields(line)
			if len(parts) != 3 {
				conn.WriteToUDP([]byte("Invalid login format.\n"), addr)
				continue
			}
			username, password := parts[1], parts[2]
			if validateUser(username, password) {
				currentUser := &UserSession{Name: username, Password: password, Addr: addr}
				mutex.Lock()
				sessions[username] = currentUser
				mutex.Unlock()
				conn.WriteToUDP([]byte("Login successful!\n"), addr)
			} else {
				conn.WriteToUDP([]byte("Invalid credentials.\n"), addr)
			}
		case strings.HasPrefix(line, "FILE"):
			parts := strings.Fields(line)
			if len(parts) < 2 {
				conn.WriteToUDP([]byte("Invalid file transfer format.\n"), addr)
				continue
			}
			handleFileTransfer(conn, addr, strings.Join(parts[1:], " "))
		case strings.HasPrefix(line, "MSG"):
			message := strings.TrimSpace(strings.TrimPrefix(line, "MSG "))
			conn.WriteToUDP([]byte("Message received: "+message+"\n"), addr)
			fmt.Println("[MSG]: " + message)

		default:
			conn.WriteToUDP([]byte("Unknown command.\n"), addr)
		}
	}
}

func validateUser(username, password string) bool {
	validUsers := map[string]string{
		"user1": "pass1",
		"user2": "pass2",
		"user3": "pass3",
	}
	if pass, ok := validUsers[username]; ok {
		return pass == password
	}
	return false
}

func handleFileTransfer(conn *net.UDPConn, addr net.Addr, fileName string) {
	fmt.Println("[FILE]")
	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			break
		}
		fmt.Printf("Received data: %s", string(buffer[:n]))
	}
	fmt.Println("File received from:", addr.String(), ":", fileName)
}

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 8080})
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}
		go handlePacket(conn, buf[:n], addr)
	}
}
