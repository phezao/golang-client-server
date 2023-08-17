package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type Cotacao struct {
	ID    string `json:"id"`
	Dolar string `json:"dolar"`
}

func NewCotacao(dolar string) *Cotacao {
	return &Cotacao{
		ID:    uuid.New().String(),
		Dolar: dolar,
	}
}

type Cambio struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	}
}

func main() {
	http.HandleFunc("/cotacao", Handler)
	http.ListenAndServe(":8080", nil)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/cotacao")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := r.Context()
	log.Println("Request iniciado")
	defer log.Println("Request finalizada")
	cotacao, err := PegaCotacao()
	if err != nil {
		log.Println("Request Timeout, erro no request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	NovaCotacao := NewCotacao(cotacao.USDBRL.Bid)
	err = insertCotacao(db, NovaCotacao)
	if err != nil {
		panic(err)
	}
	select {
	case <-time.After(200 * time.Millisecond):
		// Log printa no stdout
		log.Println("Request processada com sucesso")
		// Imprime no browser
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(NovaCotacao)
	case <-ctx.Done():
		log.Println("Request cancelada pelo cliente")
	}
}

func PegaCotacao() (*Cambio, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var data Cambio
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Valor de data: %+v", data)
	return &data, nil
}

func insertCotacao(db *sql.DB, cotacao *Cotacao) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stmt, err := db.Prepare("insert into cotacoes(id, dolar) values (?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, cotacao.ID, cotacao.Dolar)
	if err != nil {
		return err
	}
	return nil
}
