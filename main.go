package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hyperjumptech/grule-rule-engine/ast"
	"github.com/hyperjumptech/grule-rule-engine/builder"
	"github.com/hyperjumptech/grule-rule-engine/pkg"
	"github.com/rs/cors"
)

type Request struct {
	Player    []string `json:"player"`
	Dealer    []string `json:"dealer"`
	GameRules struct {
		DASAllowed       bool `json:"dasAllowed"`
		SurrenderAllowed bool `json:"surrenderAllowed"`
		Decks            int  `json:"decks"`
	} `json:"gameRules"`
}

type Response struct {
	Action string `json:"action"`
}

type BlackjackFact struct {
	PlayerTotal      int
	SoftTotal        bool
	DealerCard       int
	CanSplit         bool
	DASAllowed       bool
	SurrenderAllowed bool
	Decks            int
	RecommendedAct   string
	Insurance        bool
}

var knowledgeLibrary *ast.KnowledgeLibrary

const strategyRules = `
/* Reglas para Pares */
rule SplitAcesAnd8s "Split Aces and 8s" {
    when
        Fact.CanSplit == true &&
        (Fact.PlayerTotal == 22 || Fact.PlayerTotal == 16)
    then
        Fact.RecommendedAct = "SPLIT";
        Retract("Split Aces and 8s");
}

rule NeverSplit5sAnd10s "Never Split 5s and 10s" {
    when
        Fact.CanSplit == true &&
        (Fact.PlayerTotal == 10 || Fact.PlayerTotal == 20)
    then
        Fact.RecommendedAct = "STAND";
        Retract("Never Split 5s and 10s");
}

rule Split2s3s7s "Split 2s, 3s, 7s vs 2-7" {
    when
        Fact.CanSplit == true &&
        (Fact.PlayerTotal == 4 || Fact.PlayerTotal == 6 || Fact.PlayerTotal == 14) &&
        Fact.DealerCard >= 2 &&
        Fact.DealerCard <= 7
    then
        Fact.RecommendedAct = "SPLIT";
        Retract("Split 2s, 3s, 7s vs 2-7");
}

rule Split4s "Split 4s vs 5-6" {
    when
        Fact.CanSplit == true &&
        Fact.PlayerTotal == 8 &&
        Fact.DealerCard >= 5 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "SPLIT";
        Retract("Split 4s vs 5-6");
}

rule Split6s "Split 6s vs 2-6" {
    when
        Fact.CanSplit == true &&
        Fact.PlayerTotal == 12 &&
        Fact.DealerCard >= 2 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "SPLIT";
        Retract("Split 6s vs 2-6");
}

rule Split9s "Split 9s vs 2-9" {
    when
        Fact.CanSplit == true &&
        Fact.PlayerTotal == 18 &&
        Fact.DealerCard >= 2 &&
        Fact.DealerCard <= 9
    then
        Fact.RecommendedAct = "SPLIT";
        Retract("Split 9s vs 2-9");
}

/* Reglas para Manos Suaves (Aces) */
rule Soft13_14 "Soft 13-14 vs 5-6" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal >= 13 &&
        Fact.PlayerTotal <= 14 &&
        Fact.DealerCard >= 5 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Soft 13-14 vs 5-6");
}

rule Soft15_16 "Soft 15-16 vs 4-6" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal >= 15 &&
        Fact.PlayerTotal <= 16 &&
        Fact.DealerCard >= 4 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Soft 15-16 vs 4-6");
}

rule Soft17 "Soft 17 vs 3-6" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal == 17 &&
        Fact.DealerCard >= 3 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Soft 17 vs 3-6");
}

rule Soft18High "Soft 18 vs 9+" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal == 18 &&
        Fact.DealerCard >= 9
    then
        Fact.RecommendedAct = "HIT";
        Retract("Soft 18 vs 9+");
}

rule Soft18Mid "Soft 18 vs 2,7,8" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal == 18 &&
        (Fact.DealerCard == 2 || Fact.DealerCard == 7 || Fact.DealerCard == 8)
    then
        Fact.RecommendedAct = "STAND";
        Retract("Soft 18 vs 2,7,8");
}

rule Soft18Low "Soft 18 vs 3-6" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal == 18 &&
        Fact.DealerCard >= 3 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Soft 18 vs 3-6");
}

rule Soft19Plus "Soft 19+" {
    when
        Fact.SoftTotal == true &&
        Fact.PlayerTotal >= 19
    then
        Fact.RecommendedAct = "STAND";
        Retract("Soft 19+");
}

/* Reglas para Manos Duras */
rule Hard5_8 "Hard 5-8" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal >= 5 &&
        Fact.PlayerTotal <= 8
    then
        Fact.RecommendedAct = "HIT";
        Retract("Hard 5-8");
}

rule Hard9Double "Hard 9 Double" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 9 &&
        Fact.DealerCard >= 3 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Hard 9 Double");
}

rule Hard9Hit "Hard 9 Hit" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 9 &&
        (Fact.DealerCard < 3 || Fact.DealerCard > 6)
    then
        Fact.RecommendedAct = "HIT";
        Retract("Hard 9 Hit");
}

rule Hard10Double "Hard 10 Double" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 10 &&
        Fact.DealerCard <= 9
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Hard 10 Double");
}

rule Hard10Hit "Hard 10 Hit" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 10 &&
        Fact.DealerCard > 9
    then
        Fact.RecommendedAct = "HIT";
        Retract("Hard 10 Hit");
}

rule Hard11 "Hard 11" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 11
    then
        Fact.RecommendedAct = "DOUBLE";
        Retract("Hard 11");
}

rule Hard12Stand "Hard 12 Stand" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 12 &&
        Fact.DealerCard >= 4 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "STAND";
        Retract("Hard 12 Stand");
}

rule Hard12Hit "Hard 12 Hit" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 12 &&
        (Fact.DealerCard < 4 || Fact.DealerCard > 6)
    then
        Fact.RecommendedAct = "HIT";
        Retract("Hard 12 Hit");
}

rule Hard13_16Stand "Hard 13-16 Stand" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal >= 13 &&
        Fact.PlayerTotal <= 16 &&
        Fact.DealerCard <= 6
    then
        Fact.RecommendedAct = "STAND";
        Retract("Hard 13-16 Stand");
}

rule Hard13_16Hit "Hard 13-16 Hit" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal >= 13 &&
        Fact.PlayerTotal <= 16 &&
        Fact.DealerCard > 6
    then
        Fact.RecommendedAct = "HIT";
        Retract("Hard 13-16 Hit");
}

rule Hard17Plus "Hard 17+" {
    when
        Fact.SoftTotal == false &&
        Fact.PlayerTotal >= 17
    then
        Fact.RecommendedAct = "STAND";
        Retract("Hard 17+");
}

/* Reglas de Rendición */
rule Surrender16 "Surrender 16 vs 9-11" {
    when
        Fact.SurrenderAllowed == true &&
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 16 &&
        Fact.DealerCard >= 9
    then
        Fact.RecommendedAct = "SURRENDER";
        Retract("Surrender 16 vs 9-11");
}

rule Surrender15 "Surrender 15 vs 10" {
    when
        Fact.SurrenderAllowed == true &&
        Fact.SoftTotal == false &&
        Fact.PlayerTotal == 15 &&
        Fact.DealerCard == 10
    then
        Fact.RecommendedAct = "SURRENDER";
        Retract("Surrender 15 vs 10");
}

/* Reglas Especiales */
rule DealerAceInsurance "Dealer Ace Insurance" {
    when
        Fact.DealerCard == 11
    then
        Fact.Insurance = true;
        Retract("Dealer Ace Insurance");
}

rule BlackjackNatural "Blackjack Natural" {
    when
        Fact.PlayerTotal == 21
    then
        Fact.RecommendedAct = "BLACKJACK";
        Retract("Blackjack Natural");
}
`

var extensionID = fmt.Sprintf("chrome-extension://%s", os.Getenv("extension_id"))
var secretToken = os.Getenv("secret_token")

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == extensionID {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	})
}
func tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "Bearer "+secretToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	initRuleEngine()

	mux := http.NewServeMux()
	protected := tokenMiddleware(mux)
	secured := corsMiddleware(protected)

	http.ListenAndServe(":8080", secured)
	mux.HandleFunc("/api/advice", adviceHandler)

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
	}).Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciado y escuchando en el puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func initRuleEngine() {
	knowledgeLibrary = ast.NewKnowledgeLibrary()
	ruleBuilder := builder.NewRuleBuilder(knowledgeLibrary)

	err := ruleBuilder.BuildRuleFromResource(
		"BlackjackStrategy", // Nombre del conocimiento
		"1.0.0",             // Versión
		pkg.NewBytesResource([]byte(strategyRules)),
	)
	if err != nil {
		log.Fatal("Error building rules:", err)
	}
}

func adviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if len(req.Player) < 2 || len(req.Dealer) < 1 {
		sendError(w, "Invalid number of cards", http.StatusBadRequest)
		return
	}

	fact, err := createFact(req)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	executeRules(fact)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Action: fact.RecommendedAct})
}

func executeRules(fact *BlackjackFact) {
	if fact.PlayerTotal == 21 {
		fact.RecommendedAct = "BLACKJACK"
		return
	}

	if fact.DealerCard == 11 {
		fact.Insurance = true
	}

	if fact.CanSplit {
		if fact.PlayerTotal == 12 && fact.DealerCard >= 2 && fact.DealerCard <= 6 {
			fact.RecommendedAct = "SPLIT" // Par de 6s
			return
		}
		if fact.PlayerTotal == 14 && fact.DealerCard >= 2 && fact.DealerCard <= 7 {
			fact.RecommendedAct = "SPLIT" // Par de 7s
			return
		}
		if fact.PlayerTotal == 16 {
			fact.RecommendedAct = "SPLIT" // Par de 8s
			return
		}
		if fact.PlayerTotal == 18 && fact.DealerCard <= 9 {
			fact.RecommendedAct = "SPLIT" // Par de 9s
			return
		}
		if fact.PlayerTotal == 20 {
			fact.RecommendedAct = "STAND" // Par de 10s
			return
		}
		if fact.PlayerTotal == 22 || (fact.PlayerTotal == 12 && fact.SoftTotal) {
			fact.RecommendedAct = "SPLIT" // Par de ases
			return
		}
	}

	if fact.SurrenderAllowed {
		if fact.SoftTotal == false {
			if fact.PlayerTotal == 16 && fact.DealerCard >= 9 {
				fact.RecommendedAct = "SURRENDER"
				return
			}
			if fact.PlayerTotal == 15 && fact.DealerCard == 10 {
				fact.RecommendedAct = "SURRENDER"
				return
			}
		}
	}

	if fact.SoftTotal {
		if fact.PlayerTotal >= 19 {
			fact.RecommendedAct = "STAND"
		} else if fact.PlayerTotal == 18 {
			if fact.DealerCard >= 9 {
				fact.RecommendedAct = "HIT"
			} else if fact.DealerCard >= 3 && fact.DealerCard <= 6 {
				fact.RecommendedAct = "DOUBLE"
			} else {
				fact.RecommendedAct = "STAND"
			}
		} else if fact.PlayerTotal == 17 && fact.DealerCard >= 3 && fact.DealerCard <= 6 {
			fact.RecommendedAct = "DOUBLE"
		} else {
			fact.RecommendedAct = "HIT"
		}
		return
	}

	if fact.PlayerTotal >= 17 {
		fact.RecommendedAct = "STAND"
	} else if fact.PlayerTotal <= 8 {
		fact.RecommendedAct = "HIT"
	} else if fact.PlayerTotal == 11 {
		fact.RecommendedAct = "DOUBLE"
	} else if fact.PlayerTotal == 10 {
		if fact.DealerCard <= 9 {
			fact.RecommendedAct = "DOUBLE"
		} else {
			fact.RecommendedAct = "HIT"
		}
	} else if fact.PlayerTotal == 9 {
		if fact.DealerCard >= 3 && fact.DealerCard <= 6 {
			fact.RecommendedAct = "DOUBLE"
		} else {
			fact.RecommendedAct = "HIT"
		}
	} else if fact.PlayerTotal >= 12 && fact.PlayerTotal <= 16 {
		if fact.DealerCard >= 7 {
			fact.RecommendedAct = "HIT"
		} else {
			fact.RecommendedAct = "STAND"
		}
	}

	if fact.RecommendedAct == "" {
		fact.RecommendedAct = "HIT"
	}
}

func createFact(req Request) (*BlackjackFact, error) {
	playerTotal, soft := calculateTotal(req.Player)
	dealerCard, err := cardValue(req.Dealer[0])
	if err != nil {
		return nil, err
	}

	return &BlackjackFact{
		PlayerTotal:      playerTotal,
		SoftTotal:        soft,
		DealerCard:       dealerCard,
		CanSplit:         canSplit(req.Player),
		DASAllowed:       req.GameRules.DASAllowed,
		SurrenderAllowed: req.GameRules.SurrenderAllowed,
		Decks:            req.GameRules.Decks,
	}, nil
}

func calculateTotal(cards []string) (int, bool) {
	total := 0
	aces := 0

	for _, card := range cards {
		val, _ := cardValue(card)
		if val == 11 {
			aces++
		}
		total += val
	}

	soft := false
	for total > 21 && aces > 0 {
		total -= 10
		aces--
		soft = aces > 0
	}

	return total, soft
}

func cardValue(card string) (int, error) {
	switch strings.ToUpper(card) {
	case "A":
		return 11, nil
	case "J", "Q", "K":
		return 10, nil
	default:
		if val, err := strconv.Atoi(card); err == nil && val >= 2 && val <= 10 {
			return val, nil
		}
		return 0, fmt.Errorf("invalid card value: %s", card)
	}
}

func canSplit(cards []string) bool {
	if len(cards) != 2 {
		return false
	}
	val1, _ := cardValue(cards[0])
	val2, _ := cardValue(cards[1])
	return val1 == val2
}

func sendError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
