package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func sendFile(conn *net.UDPConn, addr *net.UDPAddr) {
	fmt.Println("Please specify the file with absolute path: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading path:", err)
		return
	}
	trimmedText := strings.TrimSpace(text)

	file, err := os.Open(trimmedText)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fileReader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	for {
		n, err := fileReader.Read(buffer)
		if err != nil {
			break
		}
		_, err = conn.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error sending data:", err)
			return
		}
	}
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Failed to connect to the server:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter command: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			break
		}

		trimmedText := strings.TrimSpace(text)
		if trimmedText == "EXIT" {
			break // 用户输入 EXIT，跳出循环
		}

		_, err = conn.Write([]byte(trimmedText))
		if err != nil {
			fmt.Println("Error sending command:", err)
			break
		}

		// 从服务器读取响应
		buffer := make([]byte, 1024)
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading response:", err)
			break
		}
		response := string(buffer[:n])
		fmt.Print(response)

		if strings.HasPrefix(response, "Ready to receive") {
			sendFile(conn, addr)
		}
	}

	fmt.Println("Exiting...")
}
