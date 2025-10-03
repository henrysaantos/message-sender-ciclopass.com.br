package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto" // <-- alias correto
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

var client *whatsmeow.Client

type MessageRequest struct {
	To   string `json:"to"`
	Text string `json:"text"`
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jid, err := types.ParseJID(req.To)
	if err != nil {
		http.Error(w, "JID inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	text := req.Text
	_, err = client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: &text,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"sent"}`))
}

func main() {
	// 🔹 Conectar SQLite
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:example.db?_foreign_keys=on", waLog.Noop)
	if err != nil {
		log.Fatalf("Erro ao abrir banco: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Erro ao obter device: %v", err)
	}

	client = whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "INFO", true))

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
					log.Println("➡️ Escaneie o QR Code acima com seu WhatsApp")
				} else {
					log.Printf("Evento QR: %v", evt.Event)
				}
			}
		}()
		log.Println("🔌 Conectando ao WhatsApp...")
		err = client.Connect()
		if err != nil {
			log.Fatalf("Erro ao conectar: %v", err)
		}
	} else {
		log.Println("🔌 Conectando ao WhatsApp com sessão salva...")
		err = client.Connect()
		if err != nil {
			log.Fatalf("Erro ao conectar: %v", err)
		}
		log.Println("✅ Já logado no WhatsApp")
	}

	http.HandleFunc("/send", sendMessageHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	log.Println("🚀 Servidor rodando em http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
