package main

import (
	"fmt"
	"os"
	"os/exec"
	"log"
	"bufio"
	"encoding/json"
)

type Account struct{
	Username string `json:"Name"`
	Password string `json:"Password"`
}

func main() {
	for {
		fmt.Println("Welcome to the Game Hub!")
		fmt.Println("Please choose a game to play:")
		fmt.Println("1. Pokecat")
		fmt.Println("2. Pokebat")
		fmt.Println("3. Exit")
		fmt.Print("Enter your choice: ")

		var choice int
		_, err := fmt.Scanln(&choice)
		if err != nil {
			fmt.Println("Invalid input. Please enter a number.")
			continue
		}

		switch choice {
		case 1:
			fmt.Println("Launching Pokecat...")
			cmd := exec.Command("go", "run", "pokecat/player.go")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Run the player.go script
			if err := cmd.Run(); err != nil {
				fmt.Printf("Failed to launch Pokecat: %v\n", err)
			}
		case 2:
			fmt.Println("Launching Pokebat...")
			cmd := exec.Command("go", "run", "pokebat/clients.go")
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Run the clients.go script
			if err := cmd.Run(); err != nil {
				fmt.Printf("Failed to launch Pokebat: %v\n", err)
			}
		case 3:
			fmt.Println("Exiting the Game Hub. Goodbye!")
			os.Exit(0)
		
		
		default:
			fmt.Println("Invalid choice. Please choose a valid option.")
		}
	}
}
