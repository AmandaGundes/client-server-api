package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Fatalf("erro criando requisição: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatalf("timeout ao chamar servidor")
		}
		log.Fatalf("erro ao chamar servidor: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("erro do servidor: status %d — %s",
			resp.StatusCode, string(errBody))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("erro ao ler resposta: %v", err)
	}
	var result struct {
		Bid string `json:"bid"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("erro ao decodificar JSON: %v", err)
	}

	line := "Dólar: " + result.Bid + "\n"
	if err := ioutil.WriteFile("cotacao.txt", []byte(line), 0644); err != nil {
		log.Fatalf("erro ao escrever arquivo: %v", err)
	}

	log.Println("Cotação salva em cotacao.txt:", line)
}
