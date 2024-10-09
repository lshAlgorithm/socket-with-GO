package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func sendFile(conn net.Conn) {
	fmt.Println("Please specify the file with absolute path: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading path:", err)
		return
	}
	// Send file content
	trimmedText := strings.TrimSpace(text) // 移除空白字符
	file, err := os.Open(trimmedText)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileReader := bufio.NewReader(file)
	for {
		data := make([]byte, 1024)
		n, err := fileReader.Read(data)
		if err != nil {
			break
		}
		_, err = conn.Write(data[:n])
		if err != nil {
			fmt.Println("Error writing data:", err)
			return
		}
	}
}

var finishPrint = make(chan bool)
var serverFail = make(chan bool)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		return
	}
	defer conn.Close()

	go func() {
		finishPrint <- true
		for {
			response, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				fmt.Println("Error reading response:", err)
				serverFail <- true
				return
			}
			fmt.Print(response)
			if strings.HasPrefix(response, "Received from another user") == false {
				finishPrint <- true
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	flag := false
	for {
		select {
		case <-serverFail:
			flag = true
		default:
			// Do nothing
		}

		if flag {
			break
		}

		if <-finishPrint {
			fmt.Print("Enter command: ")
		}
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			break
		}

		trimmedText := strings.TrimSpace(text)
		if trimmedText == "EXIT" {
			break
		}

		_, err = conn.Write([]byte(trimmedText + "\n"))
		if err != nil {
			fmt.Println("Error writing to socket:", err)
			break
		}
	}

	fmt.Println("Exiting...")

}
