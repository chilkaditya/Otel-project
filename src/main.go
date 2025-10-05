package main

import (
	"time"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
    meter = otel.GetMeterProvider().Meter("otel-tut")
    signupCounter, _ = meter.Int64Counter("signup_count")
    // Traffic
    requestCounter, _ = meter.Int64Counter("http_requests_total")
    // Errors
    errorCounter, _ = meter.Int64Counter("http_errors_total")
    // Latency
    latencyHistogram, _ = meter.Float64Histogram("http_request_duration_seconds")
    // Saturation (example: DB connections)
    dbConnections, _ = meter.Int64ObservableGauge("db_connections_active")
)

var (
	logger = otel.GetLoggerProvider().Logger("otel-tut") 
)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    rm := metricdata.ResourceMetrics{}
    if err := MetricReader.Collect(ctx, &rm); err != nil {
        http.Error(w, "Failed to collect metrics", http.StatusInternalServerError)
        return
    }
    b, err := json.MarshalIndent(rm, "", "  ")
    if err != nil {
        http.Error(w, "Failed to marshal metrics", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(b)
}


func homeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := otel.Tracer("otel-tut").Start(r.Context(), "homeHandler")
	defer span.End()

	// we can set attributes, events, status, etc in span
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("user.agent", r.UserAgent()),
		attribute.String("handler", "home"),
	)

	logger.Emit(ctx, log.Info("Received home page request",
        log.String("http.method", r.Method),
        log.String("user.agent", r.UserAgent()),
    ))

	defer func() { 
		duration := time.Since(start).Seconds() 
		latencyHistogram.Record(ctx, duration, metric.WithAttributes(attribute.String("handler", "home"))) 
	}()

	requestCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "home")))

	w.Header().Set("Content-type", "text/html")
	fmt.Fprintln(w, "<h2>Welcome to the home page</h2>")
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	//tracing
	ctx, span := otel.Tracer("otel-tut").Start(r.Context(), "signupHandler")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("user.agent", r.UserAgent()),
		attribute.String("handler", "signup"),
	)
	logger.Emit(ctx, log.Info("Received signup request",
        log.String("http.method", r.Method),
        log.String("user.agent", r.UserAgent()),
    ))

	// metrics-traffic
	requestCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "signup")))

	//metrics-latency
	defer func() {
        duration := time.Since(start).Seconds()
        latencyHistogram.Record(ctx, duration, metric.WithAttributes(attribute.String("handler", "signup")))
    }()

	if r.Method != http.MethodPost {
		//metrics-error rate
		errorCounter.Add(r.Context(), 1, 
		metric.WithAttributes(attribute.String("handler", "signup"), attribute.String("reason", "wrong method")))
		logger.Emit(ctx, log.Info("Signup request failed",
        log.String("http.method", r.Method),
        log.String("user.agent", r.UserAgent()),
    	))

		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		//metrics-error rate
		errorCounter.Add(r.Context(), 1, 
		metric.WithAttributes(attribute.String("handler", "signup"), attribute.String("reason", "wrong request fromat")))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var exists int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&exists)
	if err != nil {
		//metrics-error rate
		errorCounter.Add(r.Context(), 1, 
		metric.WithAttributes(attribute.String("handler", "signup"), attribute.String("rason", "DB error(query)")))
		http.Error(w, "Database error(query)", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		//metrics-error rate
		errorCounter.Add(r.Context(), 1, 
		metric.WithAttributes(attribute.String("handler", "signup"), attribute.String("reason", "user exists")))
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}
	_, err = DB.Exec("INSERT INTO users(username, password) VALUES(?, ?)", user.Username, user.Password)
	if err != nil {
		//metrics-error rate
		errorCounter.Add(r.Context(),1, 
		metric.WithAttributes(attribute.String("handler", "signup"), attribute.String("reason", "Db error(execute)")))
		http.Error(w, "Database error(execute)", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Signup successful")
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := otel.Tracer("otel-tut").Start(r.Context(), "signinHandler")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("user.agent", r.UserAgent()),
		attribute.String("handler", "sign"),
	)
	defer func() {
        duration := time.Since(start).Seconds()
        latencyHistogram.Record(ctx, duration, metric.WithAttributes(attribute.String("handler", "signin")))
    }()

	requestCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "signin")))

	var pass string
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		errorCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "signin"), attribute.String("reason", "invalid request")))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	err := DB.QueryRow("SELECT password FROM users WHERE username = ?", user.Username).Scan(&pass)
	if pass != user.Password {
		errorCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "signin"), attribute.String("reason", "invalid user")))
		http.Error(w, "Invalid user", http.StatusUnauthorized)
		return
	} else if err != nil {
		errorCounter.Add(r.Context(), 1, metric.WithAttributes(attribute.String("handler", "signin"), attribute.String("reason", "DB error")))
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "Signin successful")
}

func main() {
	ctx := context.Background()
	if err := InitOpenTelemetry(ctx); err != nil {
		panic(err)
	}
	if err := InitDB(); err != nil {
		panic(err)
	}
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/signin", signinHandler)
	http.HandleFunc("/metric", metricsHandler)
	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}