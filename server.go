package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

type QuoteResponse struct {
    USDBRL struct {
        Bid string `json:"bid"`
    } `json:"USDBRL"`
}

func main() {
    db, err := sql.Open("sqlite3", "cotacoes.db")
    if err != nil {
        log.Fatalf("erro ao abrir banco de dados: %v", err)
    }
    defer db.Close()

    _, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS cotacoes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        bid TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    )`)
    if err != nil {
        log.Fatalf("erro ao criar tabela: %v", err)
    }

    http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Timeout de 200ms para chamar a API externa
        ctxAPI, cancelAPI := context.WithTimeout(ctx, 200*time.Millisecond)
        defer cancelAPI()

        req, err := http.NewRequestWithContext(ctxAPI, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
        if err != nil {
            log.Printf("erro criando requisição para API: %v", err)
            http.Error(w, "erro interno", http.StatusInternalServerError)
            return
        }

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            if ctxAPI.Err() == context.DeadlineExceeded {
                log.Printf("timeout na chamada da API")
                http.Error(w, "timeout na cotação", http.StatusGatewayTimeout)
            } else {
                log.Printf("erro ao chamar API: %v", err)
                http.Error(w, "erro interno", http.StatusInternalServerError)
            }
            return
        }
        defer resp.Body.Close()

        var qr QuoteResponse
        if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
            log.Printf("erro ao decodificar resposta da API: %v", err)
            http.Error(w, "erro interno", http.StatusInternalServerError)
            return
        }
        bid := qr.USDBRL.Bid

        // Timeout de 10ms para persistir no SQLite
        ctxDB, cancelDB := context.WithTimeout(ctx, 10*time.Millisecond)
        defer cancelDB()

        if _, err := db.ExecContext(ctxDB, "INSERT INTO cotacoes(bid) VALUES(?)", bid); err != nil {
            if ctxDB.Err() == context.DeadlineExceeded {
                log.Printf("timeout ao persistir no banco")
            } else {
                log.Printf("erro ao persistir no banco: %v", err)
            }
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"bid": bid})
    })

    log.Println("Servidor iniciado em :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("erro no servidor: %v", err)
    }
}
