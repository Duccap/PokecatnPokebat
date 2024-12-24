package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)
type Account struct{
	Username string `json:"Name"`
	Password string `json:"Password"`
}

type Pokemon struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Types        []string          `json:"types"`
	Stats        Stats             `json:"stats"`
	Exp          int               `json:"exp,string"`
	WhenAttacked map[string]string `json:"when_attacked"`
}

type Stats struct {
	HP      int `json:"HP,string"`
	Attack  int `json:"Attack,string"`
	Defense int `json:"Defense,string"`
	Speed   int `json:"Speed,string"`
	SpAtk   int `json:"Sp Atk,string"`
	SpDef   int `json:"Sp Def,string"`
}

type Player struct {
	Name     string     `json:"name"`
	Pokemons []*Pokemon `json:"pokemons"`
	Active   *Pokemon   `json:"active"`
	Conn     net.Conn
}

func main() {
	// Start the server
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server started. Waiting for players...")

	players := make([]*Player, 0, 2)
	playerNames := make(map[string]bool)

	// Accept two players
	for len(players) < 2 {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		username, authenticated := authenticatePlayer(conn)
		if !authenticated {
			log.Printf("Authentication failed for connection from %s", conn.RemoteAddr())
			conn.Close()
			continue
		}

		// Check if the player is already in the battle
		if playerNames[username] {
			log.Printf("Player %s is already in the battle", username)
			conn.Write([]byte("You are already in the battle. Exiting.\n"))
			conn.Close()
			continue
		}

		// Use username as player_name to load data
		playerData, err := loadPlayerData("../player_data.json", username)
		if err != nil {
			log.Printf("Failed to load player data for %s: %v", username, err)
			conn.Write([]byte("Failed to load player data. Exiting.\n"))
			conn.Close()
			continue
		}

		// Assign the connection to the player
		playerData.Conn = conn

		// Add the player to the players list and mark the player as joined
		players = append(players, playerData)
		playerNames[username] = true
		log.Printf("Player %s has joined with their saved data.", username)

		// Notify the player
		conn.Write([]byte(fmt.Sprintf("Welcome back, %s! AWAIT THE BATTLE!!!!!\n", playerData.Name)))

		fmt.Printf("Player %d connected from %s\n", len(players), conn.RemoteAddr())
	}

	for _, player := range players {
		selectPokemons(player)
	}

	// Simplify turn order logic based on speed
	var firstPlayer, secondPlayer *Player
	if players[0].Active.Stats.Speed > players[1].Active.Stats.Speed {
		firstPlayer = players[0]
		secondPlayer = players[1]
	} else {
		firstPlayer = players[1]
		secondPlayer = players[0]
	}

	// Start battle loop
	startBattle(firstPlayer, secondPlayer)
}



// Authenticate the player using credentials from accounts.json
func authenticatePlayer(conn net.Conn) (string, bool) {
    buffer := make([]byte, 2048)
    n, err := conn.Read(buffer)
    if err != nil {
        log.Printf("Failed to read authentication data: %v", err)
        return "", false
    }

    var authData map[string]string
    if err := json.Unmarshal(buffer[:n], &authData); err != nil {
        log.Printf("Failed to parse authentication data: %v", err)
        return "", false
    }

    accounts, err := loadAccountsData("../accounts.json")
    if err != nil {
        log.Printf("Failed to load accounts data: %v", err)
        return "", false
    }

    // Authenticate the user by iterating over all accounts
    for _, account := range accounts {
        if account.Username == authData["name"] && account.Password == authData["password"] {
            response := map[string]string{"status": "success"}
            responseBytes, _ := json.Marshal(response)
            conn.Write(responseBytes)
            return authData["name"], true
        }
    }

    response := map[string]string{"status": "failure"}
    responseBytes, _ := json.Marshal(response)
    conn.Write(responseBytes)
    return "", false
}



// Load accounts data from accounts.json
func loadAccountsData(filename string) ([]Account, error) {
    file, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to load accounts data file: %v", err)
    }

    var accounts []Account
    if err := json.Unmarshal(file, &accounts); err != nil {
        return nil, fmt.Errorf("failed to parse accounts data: %v", err)
    }

    log.Printf("Loaded %d accounts from %s", len(accounts), filename)
    return accounts, nil
}



// Load player data from player_data.json
func loadPlayerData(filename, playerName string) (*Player, error) {
    file, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to load player_data.json: %v", err)
    }

    var playerDatas []map[string]interface{}
    if err := json.Unmarshal(file, &playerDatas); err != nil {
        return nil, fmt.Errorf("failed to parse player_data.json: %v", err)
    }

    // Look for the player data based on player_name
    for _, playerData := range playerDatas {
        if playerData["player_name"] == playerName {
            // Parse the Pokémon data
            pokemonsData, _ := json.Marshal(playerData["pokemons"])
            var pokemons []*Pokemon
            if err := json.Unmarshal(pokemonsData, &pokemons); err != nil {
                return nil, fmt.Errorf("failed to parse pokemons data: %v", err)
            }

            return &Player{
                Name:     playerName,
                Pokemons: pokemons,
                Active:   pokemons[0], // Set the first Pokémon as active
            }, nil
        }
    }

    return nil, fmt.Errorf("player data not found for player_name: %s", playerName)
}

func selectPokemons(player *Player) {
	if len(player.Pokemons) < 3 {
		player.Conn.Write([]byte("You need at least 3 Pokémon to battle. Please play PokéCat to catch more Pokémon.\n"))
		player.Conn.Close()
		return
	}

	for {
		player.Conn.Write([]byte("Here are your available Pokémon:\n"))
		for i, pokemon := range player.Pokemons {
			// Format the types to uppercase
			types := strings.ToUpper(strings.Join(pokemon.Types, ", "))

			// Create bar representations of the stats
			hpBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.HP)/10)))
			attackBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.Attack)/10)))
			defenseBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.Defense)/10)))
			speedBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.Speed)/10)))
			spAtkBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.SpAtk)/10)))
			spDefBar := strings.Repeat("🟩", int(math.Ceil(float64(pokemon.Stats.SpDef)/10)))

			// Send the formatted Pokémon details to the player
			player.Conn.Write([]byte(fmt.Sprintf(
				"%d. %s (ID: %s)\nType: %s\nHP:      %s\nAttack:  %s\nDefense: %s\nSpeed:   %s\nSp Atk:  %s\nSp Def:  %s\n\n",
				i+1, pokemon.Name, pokemon.ID, types, hpBar, attackBar, defenseBar, speedBar, spAtkBar, spDefBar,
			)))
		}
		player.Conn.Write([]byte("Choose 3 Pokémon by entering their numbers (separated by space): "))
		choice := make([]byte, 1024)
		n, err := player.Conn.Read(choice)
		if err != nil {
			log.Printf("Failed to read Pokémon choice: %v", err)
			continue
		}
		choices := strings.Fields(string(choice[:n]))

		if len(choices) != 3 {
			player.Conn.Write([]byte("Invalid Pokémon selection. Please select exactly 3 Pokémon.\n"))
			continue
		}

		// Clear previous Pokémon selections
		selectedPokemons := make([]*Pokemon, 0, 3)

		// Check if the selected Pokémon numbers are valid
		validSelection := true
		for _, choiceNum := range choices {
			index, err := strconv.Atoi(choiceNum)
			if err != nil || index < 1 || index > len(player.Pokemons) {
				player.Conn.Write([]byte(fmt.Sprintf("Invalid choice number: %s. Please try again.\n", choiceNum)))
				validSelection = false
				break
			}
			selectedPokemons = append(selectedPokemons, player.Pokemons[index-1])
		}

		if validSelection {
			player.Pokemons = selectedPokemons
			player.Active = player.Pokemons[0] // Set the first Pokémon as active
			break
		}
	}
}


// Start battle between players
func startBattle(firstPlayer, secondPlayer *Player) {
    firstPlayer.Conn.Write([]byte(fmt.Sprintf("%s, prepare for battle!\n", firstPlayer.Name)))
    secondPlayer.Conn.Write([]byte(fmt.Sprintf("%s, prepare for battle!\n", secondPlayer.Name)))
    rand.New(rand.NewSource(time.Now().UnixNano()))

    for {
        for _, player := range []*Player{firstPlayer, secondPlayer} {
            player.Conn.Write([]byte(fmt.Sprintf("Active Pokémon: %s\n", player.Active.Name)))
            player.Conn.Write([]byte("Choose action:\n1. Attack\n2. Switch Pokémon\nEnter your choice: "))

            choice := make([]byte, 1024)
            n, err := player.Conn.Read(choice)
            if err != nil {
                log.Printf("Failed to read player choice: %v", err)
                continue
            }

            switch strings.TrimSpace(string(choice[:n])) {
            case "1":
				element := player.Active.Types[0]
				damage, attackType := calculateDamage(player.Active, secondPlayer.Active, element)
				secondPlayer.Active.Stats.HP -= damage
			
				player.Conn.Write([]byte(fmt.Sprintf("You used a %s attack! Damage dealt: %d\n", attackType, damage)))
				secondPlayer.Conn.Write([]byte(fmt.Sprintf("You received a %s attack! Damage taken: %d\n", attackType, damage)))
			
				// Print remaining HP for both players
				player.Conn.Write([]byte(fmt.Sprintf("Opponent's Pokémon HP left: %d\n", secondPlayer.Active.Stats.HP)))
				secondPlayer.Conn.Write([]byte(fmt.Sprintf("Your Pokémon HP left: %d\n", secondPlayer.Active.Stats.HP)))
			
				if secondPlayer.Active.Stats.HP <= 0 {
					secondPlayer.Conn.Write([]byte("Your Pokémon fainted!\n"))
					if allPokemonFainted(secondPlayer) {
						player.Conn.Write([]byte("You win!\n"))
						secondPlayer.Conn.Write([]byte("You lose!\n"))
						return
					}
					switchPokemon(secondPlayer)
				}
			
            case "2":
                switchPokemon(player)
            default:
                player.Conn.Write([]byte("Invalid choice. Try again.\n"))
            }

            // Switch turns
            firstPlayer, secondPlayer = secondPlayer, firstPlayer
        }
    }
}

func calculateDamage(attacker, defender *Pokemon, element string) (int, string) {
    rand := rand.New(rand.NewSource(time.Now().UnixNano()))

    // 60% chance for normal attack, 40% for special attack
    isSpecial := rand.Intn(100) < 40
    var damage int
    attackType := "normal"

    if isSpecial {
        // Special attack damage
        elementalMultiplier := getElementalMultiplier(element, defender.WhenAttacked)
        damage = int(float64(attacker.Stats.SpAtk) * elementalMultiplier) - defender.Stats.SpDef
        attackType = "special"
    } else {
        // Normal attack damage
        damage = attacker.Stats.Attack - defender.Stats.Defense
    }

    // Ensure damage is not negative
    if damage < 0 {
        damage = 0
    }

    return damage, attackType
}


func getElementalMultiplier(element string, multipliers map[string]string) float64 {
	multiplierStr, exists := multipliers[element]
	if !exists {
		return 1.0 // Default multiplier
	}

	var multiplier float64
	fmt.Sscanf(multiplierStr, "%fx", &multiplier)
	return multiplier
}

func switchPokemon(player *Player) {
	player.Conn.Write([]byte("Choose a Pokémon to switch to:\n"))
	for i, pokemon := range player.Pokemons {
		if pokemon != player.Active && pokemon.Stats.HP > 0 {
			player.Conn.Write([]byte(fmt.Sprintf("%d. %s\n", i+1, pokemon.Name)))
		}
	}

	choice := make([]byte, 1024)
	n, err := player.Conn.Read(choice)
	if err != nil {
		log.Printf("Failed to read Pokémon switch choice: %v", err)
		return
	}

	selectedIndex, err := strconv.Atoi(strings.TrimSpace(string(choice[:n])))
	if err != nil || selectedIndex < 1 || selectedIndex > len(player.Pokemons) || player.Pokemons[selectedIndex-1] == player.Active {
		player.Conn.Write([]byte("Invalid choice. Try again.\n"))
		switchPokemon(player)
		return
	}

	player.Active = player.Pokemons[selectedIndex-1]
	player.Conn.Write([]byte(fmt.Sprintf("Switched to %s\n", player.Active.Name)))
}

func allPokemonFainted(player *Player) bool {
	for _, pokemon := range player.Pokemons {
		if pokemon.Stats.HP > 0 {
			return false
		}
	}
	return true
}
