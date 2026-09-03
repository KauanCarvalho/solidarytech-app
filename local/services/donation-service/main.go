package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Donation struct {
	ID        int       `json:"id"`
	NgoID     int       `json:"ngo_id"`
	Amount    float64   `json:"amount"`
	DonorName string    `json:"donor_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type App struct {
	DB          *sql.DB
	SqsSvc      *sqs.SQS
	SqsQueueURL string
}

func main() {
	_ = godotenv.Load()

	shutdownTracer, err := initTracer("donation-service")
	if err != nil {
		log.Printf("Aviso: OTel tracer não inicializado: %v", err)
	} else {
		defer shutdownTracer(context.Background())
	}

	shutdownMeter, err := initMeter("donation-service")
	if err != nil {
		log.Printf("Aviso: OTel meter não inicializado: %v", err)
	} else {
		defer shutdownMeter(context.Background())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL é obrigatória")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil || db.Ping() != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	log.Println("Conectado ao PostgreSQL (donation-service).")

	var sqsSvc *sqs.SQS
	queueURL := os.Getenv("AWS_SQS_URL")
	region := os.Getenv("AWS_REGION")
	if queueURL != "" && region != "" {
		cfg := &aws.Config{Region: aws.String(region)}
		if endpoint := os.Getenv("AWS_ENDPOINT_URL"); endpoint != "" {
			// Setado apenas em dev (LocalStack); ausente em prod, onde o SDK usa o endpoint real da AWS.
			cfg.Endpoint = aws.String(endpoint)
		}
		sess, _ := session.NewSession(cfg)
		sqsSvc = sqs.New(sess)
		log.Println("Integração com AWS SQS ativada.")
	}

	app := &App{DB: db, SqsSvc: sqsSvc, SqsQueueURL: queueURL}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.HealthHandler)
	mux.HandleFunc("/donations", app.DonationHandler)

	metricsRecorder, err := newHTTPMetrics("donation-service")
	if err != nil {
		log.Fatalf("Erro ao inicializar métricas HTTP: %v", err)
	}

	otelHandler := otelhttp.NewHandler(mux, "donation-service-http")
	handler := metricsRecorder.middleware(otelHandler)

	log.Printf("donation-service rodando na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func (a *App) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"donation-service"}`))
}

func (a *App) DonationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var d Donation
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			http.Error(w, `{"error":"Payload inválido"}`, http.StatusBadRequest)
			return
		}

		d.Status = "APPROVED" // Simulação de gateway de pagamento
		err := a.DB.QueryRow(
			"INSERT INTO donations (ngo_id, amount, donor_name, status) VALUES ($1, $2, $3, $4) RETURNING id, created_at",
			d.NgoID, d.Amount, d.DonorName, d.Status,
		).Scan(&d.ID, &d.CreatedAt)

		if err != nil {
			log.Printf("Erro ao salvar doação: %v", err)
			http.Error(w, `{"error":"Erro interno"}`, http.StatusInternalServerError)
			return
		}

		if a.SqsSvc != nil {
			go a.sendNotificationEvent(d)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(d)
		return
	}

	if r.Method == http.MethodGet {
		rows, err := a.DB.Query("SELECT id, ngo_id, amount, donor_name, status, created_at FROM donations ORDER BY id DESC")
		if err != nil {
			http.Error(w, `{"error":"Erro interno"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		donations := []Donation{}
		for rows.Next() {
			var d Donation
			rows.Scan(&d.ID, &d.NgoID, &d.Amount, &d.DonorName, &d.Status, &d.CreatedAt)
			donations = append(donations, d)
		}

		json.NewEncoder(w).Encode(donations)
		return
	}

	http.Error(w, `{"error":"Método não permitido"}`, http.StatusMethodNotAllowed)
}

func (a *App) sendNotificationEvent(d Donation) {
	body, _ := json.Marshal(d)
	_, err := a.SqsSvc.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(body)),
		QueueUrl:    aws.String(a.SqsQueueURL),
	})
	if err != nil {
		log.Printf("Falha ao despachar evento SQS: %v", err)
	}
}
