package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	dem "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs"
	events "github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)


type RoundStart struct {
    Round       int `json:"round"`
    IsWarmup    bool `json:"is_warmup"`
    Tick        int `json:"tick"`
}

// General struct for position data
type Position struct {
    X float32 `json:"x"`
    Y float32 `json:"y"`
    Z float32 `json:"z"`
}

// KillEvent represents a kill in the game
type KillEvent struct {
    Killer        string   `json:"killer"`
    Assister      string   `json:"assister,omitempty"`
    Victim        string   `json:"victim"`
    Weapon        string   `json:"weapon"`
    Headshot      bool     `json:"headshot"`
    Penetrated    bool     `json:"penetrated"`
    Tick          int      `json:"tick"`
    KillerPos     Position `json:"killer_pos"`
    VictimPos     Position `json:"victim_pos"`
}

// GrenadeEvent represents a grenade event (thrown, exploded)
type GrenadeEvent struct {
    Thrower       string   `json:"thrower"`
    GrenadeType   string   `json:"grenade_type"`
    Position      Position `json:"position"`
    Tick          int      `json:"tick"`
}

// PlayerHurtEvent represents when a player is hurt
type PlayerHurtEvent struct {
    Player        string   `json:"player"`
    Attacker      string   `json:"attacker"`
    Health        int      `json:"health"`
    Armor         int      `json:"armor"`
    Weapon        string   `json:"weapon"`
    Damage        int      `json:"damage"`
    DamageArmor   int      `json:"damage_armor"`
    HitGroup      string   `json:"hit_group"`
    Tick          int      `json:"tick"`
}

// BombEvent represents bomb-related events (plant, defuse, explode)
type BombEvent struct {
    Player        string   `json:"player"`
    Site          string   `json:"site"`
    EventType     string   `json:"event_type"` // "plant", "defuse", "explode"
    Position      Position `json:"position"`
    Tick          int      `json:"tick"`
}

// RoundEvent represents round start and end events
type RoundEvent struct {
    EventType     string   `json:"event_type"` // "start", "end"
    Reason        string   `json:"reason,omitempty"`
    Winner        string   `json:"winner,omitempty"`
    ScoreCT       int      `json:"score_ct"`
    ScoreT        int      `json:"score_t"`
    Tick          int      `json:"tick"`
}

// Other event types can be added here...

// GameEvents to aggregate all events
type GameEvents struct {
    Kills         []KillEvent       `json:"kills"`
    Grenades      []GrenadeEvent    `json:"grenades"`
    PlayerHurts   []PlayerHurtEvent `json:"player_hurts"`
    BombEvents    []BombEvent       `json:"bomb_events"`
    RoundEvents   []RoundEvent      `json:"round_events"`
    Rounds map[int][]KillEvent `json:"rounds"`
    // Include slices for other event types...
}

func parse(demoPath string) (*GameEvents, error) {
	f, err := os.Open(demoPath)
	if err != nil {
			return nil, err
	}
	defer f.Close()

	p := dem.NewParser(f)
	defer p.Close()

	gameEvents := &GameEvents{
			Rounds: make(map[int][]KillEvent),
	}

	currentRound := 0
	isWarmup := true

	// Handler to track the start of each round and whether it's a warmup
	p.RegisterEventHandler(func(e events.RoundStart) {
			currentRound++
			isWarmup = p.GameState().IsWarmupPeriod()
	})

    // Handler for Kill events
    p.RegisterEventHandler(func(e events.Kill) {
        if isWarmup {
            return // Skip kills during warmup
        }

        var killerPos, victimPos Position
        var killerName, victimName, assisterName, weapon string

        if e.Killer != nil {
            killerPosition := e.Killer.Position()
            killerPos = Position{
                X: float32(killerPosition.X),
                Y: float32(killerPosition.Y),
                Z: float32(killerPosition.Z),
            }
            killerName = e.Killer.Name
        }

        if e.Victim != nil {
            victimPosition := e.Victim.Position()
            victimPos = Position{
                X: float32(victimPosition.X),
                Y: float32(victimPosition.Y),
                Z: float32(victimPosition.Z),
            }
            victimName = e.Victim.Name
        }

        if e.Assister != nil {
            assisterName = e.Assister.Name
        }

        if e.Weapon != nil {
            weapon = e.Weapon.String()
        }

        // Add the kill event to the current round
        gameEvents.Rounds[currentRound] = append(gameEvents.Rounds[currentRound], KillEvent{
            Killer:     killerName,
            Assister:   assisterName,
            Victim:     victimName,
            Weapon:     weapon,
            Headshot:   e.IsHeadshot,
            Penetrated: e.PenetratedObjects > 0,
            Tick:       p.CurrentFrame(),
            KillerPos:  killerPos,
            VictimPos:  victimPos,
        })
    })

	// Parse the demo to the end
	err = p.ParseToEnd()
	if err != nil {
			return nil, err
	}

	return gameEvents, nil
}

func Parsedemo(w http.ResponseWriter, r *http.Request) {
    demoPath := "https://utfs.io/f/bb4bbd6d-5291-4f77-8dcf-04606f680c0f-3ke0cr.dem" // Replace with the actual demo file path
    events, err := parse(demoPath)
    if err != nil {
        log.Fatalf("Error parsing demo: %v", err)
    }

    jsonData, err := json.MarshalIndent(events, "", "    ")
    if err != nil {
        log.Fatalf("Error marshalling JSON: %v", err)
    }

		// Creating the JSON file name from the demoPath
    baseName := strings.TrimSuffix(filepath.Base(demoPath), filepath.Ext(demoPath))
    jsonFilePath := filepath.Join(filepath.Dir(demoPath), baseName+".json")

    // Writing to the JSON file
    jsonFile, err := os.Create(jsonFilePath)
    if err != nil {
        log.Fatalf("Error creating JSON file: %v", err)
    }
    defer jsonFile.Close()

    if _, err := jsonFile.Write(jsonData); err != nil {
        log.Fatalf("Error writing to JSON file: %v", err)
    }

	jsonResp, err := json.Marshal(jsonData)
	if err != nil {
		fmt.Println("Error happened in JSON marshal. Err:", err)
	} else {
		w.Write(jsonResp)
	}

}
