package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type UserSession struct {
	Name     string
	Password string
	Channel  chan string
	Conn     net.Conn
	LoggedIn bool
}

var mutex = &sync.Mutex{}
var sessions = make(map[string]*UserSession)
var userChannels = make(map[string]chan string)

func IsUserLoggedIn(username string) bool {
	mutex.Lock()
	defer mutex.Unlock()
	if session, exits := sessions[username]; exits {
		return session.LoggedIn
	}
	return false
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	var currentUser *UserSession

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println("I get the MSG: ", line)
		switch {
		case line == "EXIT":
			conn.Write([]byte("Closing connection...\n"))
			return
		case strings.HasPrefix(line, "LOGIN"):
			if currentUser != nil {
				conn.Write([]byte("Already logged in as " + currentUser.Name + ".\n"))
				continue
			}

			parts := strings.Fields(line)
			if len(parts) != 3 {
				conn.Write([]byte("Invalid login format.\n"))
				continue
			}
			username, password := parts[1], parts[2]
			if validateUser(username, password) {
				currentUser = &UserSession{Name: username, Password: password, Conn: conn}
				currentUser.Channel = make(chan string)
				mutex.Lock()
				userChannels[username] = currentUser.Channel
				sessions[username] = currentUser
				mutex.Unlock()
				go userComm(conn, currentUser)
				conn.Write([]byte("Login successful!\n"))
			} else {
				conn.Write([]byte("Invalid credentials.\n"))
			}
		case strings.HasPrefix(line, "SWITCH"):
			// You'd better not use this function. Need upgrade...
			if currentUser == nil {
				conn.Write([]byte("Please log in first.\n"))
				continue
			}
			parts := strings.Fields(line)
			if len(parts) != 3 {
				conn.Write([]byte("Invalid switch format.\n"))
				continue
			}
			targetUsername, targetPassword := parts[1], parts[2]
			if validateUser(targetUsername, targetPassword) {
				currentUser.Name = targetUsername
				currentUser.Password = targetPassword
				conn.Write([]byte("Switched to " + targetUsername + ".\n"))
			} else {
				conn.Write([]byte("Invalid credentials for switch.\n"))
			}
		case strings.HasPrefix(line, "FILE"):
			if currentUser == nil {
				conn.Write([]byte("Please log in first.\n"))
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 2 {
				conn.Write([]byte("Invalid file transfer format.\n"))
				continue
			}
			handleFileTransfer(conn, currentUser, strings.Join(parts[1:], " "))

		case strings.HasPrefix(line, "SEND"):
			if currentUser == nil {
				conn.Write([]byte("Please log in first.\n"))
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 3 {
				conn.Write([]byte("Invalid send format. Usage: SEND <username> <message>\n"))
				continue
			}
			recipient := parts[1]
			message := strings.Join(parts[2:], " ")

			if recipientChannel, exists := userChannels[recipient]; exists {
				fmt.Println("Here is the recipient...")
				recipientChannel <- message
				fmt.Println("I sent it!!!")
				conn.Write([]byte("Message sent to " + recipient + ".\n"))
			} else {
				conn.Write([]byte("Recipient not found.\n"))
			}

		case strings.HasPrefix(line, "MSG"):
			if currentUser == nil {
				conn.Write([]byte("Please log in first.\n"))
				continue
			}
			message := strings.TrimSpace(strings.TrimPrefix(line, "MSG "))
			conn.Write([]byte("Message received: " + message + "\n"))
			fmt.Println("[MSG]: " + message)
		default:
			conn.Write([]byte("Unknown command.\n"))
		}
	}
}

func userComm(conn net.Conn, currentUser *UserSession) {
	for {
		if currentUser != nil {
			select {
			case msg, ok := <-currentUser.Channel:
				fmt.Print("I get to work for ", currentUser.Name)
				if !ok {
					return
				}
				conn.Write([]byte("Received from another user: " + msg + "\n"))
			}
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

func handleFileTransfer(conn net.Conn, currentUser *UserSession, fileName string) {
	mutex.Lock()
	session, ok := getSessionByConn(conn)
	mutex.Unlock()

	if !ok {
		conn.Write([]byte("Not logged in\n"))
		return
	}

	fmt.Fprintf(conn, "Ready to receive %s\n", fileName)

	// Create the file with the username as part of the path
	dirPath := fmt.Sprintf("../received_files/%s", session.Name)
	os.MkdirAll(dirPath, os.ModePerm)
	file, err := os.Create(fmt.Sprintf("%s/%s", dirPath, fileName))
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	fmt.Println("[FILE]")
	reader := bufio.NewReader(conn)
	for {
		data := make([]byte, 1024)
		n, err := reader.Read(data)
		if err != nil {
			break
		}
		file.Write(data[:n])
		fmt.Printf("Received data: %s", string(data[:n]))
	}
	fmt.Println("File received from", currentUser.Name, ":", fileName)
}

func getSessionByConn(conn net.Conn) (*UserSession, bool) {
	for _, session := range sessions {
		if session.Conn == conn {
			return session, true
		}
	}
	return nil, false
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Failed to start server:", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		// conn.SetDeadline(time.Now().Add(60 * time.Second))
		go handleConnection(conn)
	}
}
