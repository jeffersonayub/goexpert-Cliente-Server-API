package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Cotacao struct {
	USDBRL struct {
		Code      string `json:"code"`
		Codein    string `json:"codein"`
		Name      string `json:"name"`
		High      string `json:"high"`
		Low       string `json:"low"`
		VarBid    string `json:"varBid"`
		PctChange string `json:"pctChange"`
		Bid       string `json:"bid"`
		Ask       string `json:"ask"`
	} `json:"USDBRL"`
}

func main() {
	http.HandleFunc("/cotacao", cotacaoHandler)
	http.ListenAndServe(":8080", nil)
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	cotacao, err := getCotacao()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Panic()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(cotacao.USDBRL.Bid)
}

func getCotacao() (*Cotacao, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Tempo de execução insuficiente para a chamada da API")
		}
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cotacao Cotacao
	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		return nil, err
	}

	err = saveDatabase(&cotacao)
	if err != nil {
		return nil, err
	}

	return &cotacao, nil
}

func saveDatabase(cotacao *Cotacao) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	db, err := sql.Open("sqlite3", "./cotacoes.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS cotacoes (
			id INTEGER PRIMARY KEY,
			code TEXT,
			codein TEXT,
			name TEXT,
			high TEXT,
			low TEXT,
			varBid TEXT, 
			pctChange TEXT,
			bid TEXT,
			ask TEXT,
			created_at DATETIME DEFAULT (DATETIME(CURRENT_TIMESTAMP, 'LOCALTIME'))
		);
	`)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Tempo de execução insuficiente na criação da tabela")
		}
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO cotacoes (code, codein, name, high, low, varBid, pctChange, bid, ask)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, cotacao.USDBRL.Code, cotacao.USDBRL.Codein,
		cotacao.USDBRL.Name, cotacao.USDBRL.High,
		cotacao.USDBRL.Low, cotacao.USDBRL.VarBid,
		cotacao.USDBRL.PctChange, cotacao.USDBRL.Bid, cotacao.USDBRL.Ask)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Tempo de execução insuficiente ao inserir registro no banco de dados")
		}
		return err
	}

	return nil
}
