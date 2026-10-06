package apps

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	apierror "github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	auth_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/auth"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/handler"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/middlewares"
	"github.com/MamangRust/microservice-point-of-sale-pkg/auth"
	"github.com/MamangRust/microservice-point-of-sale-pkg/dotenv"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	otel_pkg "github.com/MamangRust/microservice-point-of-sale-pkg/otel"
	"github.com/MamangRust/microservice-point-of-sale-pkg/resilience"
	"github.com/MamangRust/microservice-point-of-sale-pkg/server"
	"github.com/MamangRust/microservice-point-of-sale-pkg/upload_image"
	"github.com/MamangRust/microservice-point-of-sale-shared/cache"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/MamangRust/microservice-point-of-sale-shared/observability"
	"github.com/go-chi/chi/v5"
	chi_mw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// allowedOrigins is the CORS whitelist previously configured through the echo
// CORS middleware.
var allowedOrigins = []string{"http://localhost:1420", "http://localhost:33451"}

type ServiceAddresses struct {
	Auth        string
	Role        string
	User        string
	Category    string
	Cashier     string
	Merchant    string
	OrderItem   string
	Order       string
	Product     string
	Transaction string
	StatsReader string
}

func loadServiceAddresses() ServiceAddresses {
	return ServiceAddresses{
		Auth:        getEnvOrDefault("GRPC_AUTH_ADDR", "localhost:50051"),
		Role:        getEnvOrDefault("GRPC_ROLE_ADDR", "localhost:50052"),
		User:        getEnvOrDefault("GRPC_USER_ADDR", "localhost:50053"),
		Cashier:     getEnvOrDefault("GRPC_CASHIER_ADDR", "localhost:50055"),
		Category:    getEnvOrDefault("GRPC_CATEGORY_ADDR", "localhost:50054"),
		Merchant:    getEnvOrDefault("GRPC_MERCHANT_ADDR", "localhost:50056"),
		OrderItem:   getEnvOrDefault("GRPC_ORDERITEM_ADDR", "localhost:50057"),
		Order:       getEnvOrDefault("GRPC_ORDER_ADDR", "localhost:50058"),
		Product:     getEnvOrDefault("GRPC_PRODUCT_ADDR", "localhost:50059"),
		Transaction: getEnvOrDefault("GRPC_TRANSACTION_ADDR", "localhost:50060"),
		StatsReader: getEnvOrDefault("GRPC_STATS_READER_ADDR", "localhost:50070"),
	}
}

func createServiceConnections(addresses ServiceAddresses, logger logger.LoggerInterface) (handler.ServiceConnections, error) {
	var connections handler.ServiceConnections

	conns := map[string]*string{
		"Auth":        &addresses.Auth,
		"Role":        &addresses.Role,
		"User":        &addresses.User,
		"Cashier":     &addresses.Cashier,
		"Category":    &addresses.Category,
		"Merchant":    &addresses.Merchant,
		"OrderItem":   &addresses.OrderItem,
		"Order":       &addresses.Order,
		"Product":     &addresses.Product,
		"Transaction": &addresses.Transaction,
		"StatsReader": &addresses.StatsReader,
	}

	// One shared interceptor across every downstream connection: each gRPC
	// method gets its own breaker/bulkhead (keyed by full method name), so a
	// slow or failing service cannot exhaust the gateway's goroutines or open
	// the circuit for unrelated services.
	guard := resilience.NewDependencyGuardInterceptor(logger)

	for name, addr := range conns {
		conn, err := createConnection(*addr, name, logger, guard)
		if err != nil {
			return connections, err
		}
		switch name {
		case "Auth":
			connections.Auth = conn
		case "Role":
			connections.Role = conn
		case "User":
			connections.User = conn
		case "Cashier":
			connections.Cashier = conn
		case "Category":
			connections.Category = conn
		case "Merchant":
			connections.Merchant = conn
		case "OrderItem":
			connections.OrderItem = conn
		case "Order":
			connections.Order = conn
		case "Product":
			connections.Product = conn
		case "Transaction":
			connections.Transaction = conn
		case "StatsReader":
			connections.StatsReader = conn
		}
	}

	return connections, nil
}

// @title PointOfsale gRPC
// @version 1.0
// @description gRPC based Point Of Sale service

// @host localhost:5000
// @BasePath /api/

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token obtained from login
// @security ApiKeyAuth
type Client struct {
	Router chi.Router
	Server *http.Server
	Logger logger.LoggerInterface
}

func (c *Client) Shutdown(ctx context.Context) error {
	return c.Server.Shutdown(ctx)
}

func RunClient() (*Client, func(), error) {
	flag.Parse()

	if err := dotenv.Viper(); err != nil {
		return nil, nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	endpoint := viper.GetString("OTEL_ENDPOINT")
	if endpoint == "" {
		endpoint = "otel-collector:4317"
	}

	// Telemetry must be initialized before the logger: the logger attaches an
	// OTLP log bridge to the LoggerProvider (otelzap).
	telemetry := otel_pkg.NewTelemetry(otel_pkg.Config{
		ServiceName:            "apigateway",
		ServiceVersion:         "1.0.0",
		Environment:            viper.GetString("ENV"),
		Endpoint:               endpoint,
		Insecure:               true,
		EnableRuntimeMetrics:   true,
		RuntimeMetricsInterval: 15 * time.Second,
		Disabled:               os.Getenv("OTEL_ENABLED") == "false",
	})
	if err := telemetry.Init(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	log, err := logger.NewLogger("apigateway", telemetry.GetLogger())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create logger: %w", err)
	}

	addresses := loadServiceAddresses()

	log.Debug("Creating gRPC connections...")
	conns, err := createServiceConnections(addresses, log)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect services: %w", err)
	}

	r := setupRouter()

	token, err := auth.NewManager(viper.GetString("SECRET_KEY"))
	if err != nil {
		log.Fatal("Failed to create token manager", zap.Error(err))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", viper.GetString("REDIS_HOST"), viper.GetString("REDIS_PORT")),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       viper.GetInt("REDIS_DB"),
	})

	cacheMetrics, err := observability.NewCacheMetrics("apigateway")
	if err != nil {
		log.Fatal("Failed to initialize cache metrics", zap.Error(err))
	}

	cacheStore := cache.NewCacheStore(redisClient, log, cacheMetrics)
	authMencache := auth_cache.NewMencache(cacheStore)
	gatewayCache := gateway_cache.NewGatewayCache(cacheStore)

	mapping := response_api.NewResponseApiMapper()
	image_upload := upload_image.NewImageUpload(log)

	obs := observability.NewTraceLoggerObservability(log)
	apiHandler := apierror.NewApiHandler(obs, log)

	depsHandler := &handler.Deps{
		Token:              token,
		Router:             r,
		Logger:             log,
		Mapping:            mapping,
		ImageUpload:        image_upload,
		ServiceConnections: &conns,
		ApiHandler:         apiHandler,
		AuthCache:          authMencache,
		GatewayCache:       gatewayCache,
	}

	handler.NewHandler(depsHandler)

	srv := &http.Server{
		Addr:              ":5000",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Info("Starting API Gateway server on :5000")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server error", zap.Error(err))
		}
	}()

	go func() {
		defer wg.Done()
		log.Info("Starting Prometheus metrics server on :8091")
		if err := http.ListenAndServe(":8091", promhttp.Handler()); err != nil {
			log.Fatal("Metrics server error", zap.Error(err))
		}
	}()

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Info("Shutting down API Gateway...")
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("HTTP server shutdown failed", zap.Error(err))
		}

		closeConnections(conns, log)

		if err := telemetry.Shutdown(context.Background()); err != nil {
			log.Error("Telemetry shutdown failed", zap.Error(err))
		}
	}

	return &Client{Router: r, Server: srv, Logger: log}, shutdown, nil
}

func setupRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(chi_mw.Recoverer)
	r.Use(chi_mw.RequestID)
	r.Use(createLoggerMiddleware())

	// Extract incoming W3C trace context (traceparent/tracestate) from HTTP
	// headers so the trace continues from NGINX/client through the gateway.
	r.Use(middlewares.TraceContextMiddleware())

	r.Use(createCORSMiddleware(allowedOrigins))
	r.Use(chi_mw.Compress(5))
	r.Use(createSecureMiddleware())

	r.Use(middlewares.JWTAuth())

	// Chi returns an empty 404 body by default. Return an explicit JSON 404 so
	// API tests and clients cannot mistake an unknown route for a successful
	// endpoint.
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.JSON(w, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": "Not Found",
			"code":    http.StatusNotFound,
		})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		_ = httpx.JSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
			"status":  "error",
			"message": "Method Not Allowed",
			"code":    http.StatusMethodNotAllowed,
		})
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	return r
}

// jsonLogFormatter renders one JSON line per request, keeping the same field
// layout as the former echo LoggerWithConfig output.
type jsonLogFormatter struct{}

type jsonLogEntry struct {
	start    time.Time
	reqID    string
	remoteIP string
	host     string
	method   string
	uri      string
	bytesIn  int64
}

func (f jsonLogFormatter) NewLogEntry(r *http.Request) chi_mw.LogEntry {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &jsonLogEntry{
		start:    time.Now(),
		reqID:    chi_mw.GetReqID(r.Context()),
		remoteIP: httpx.RealIP(r),
		host:     r.Host,
		method:   r.Method,
		uri:      fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI),
		bytesIn:  r.ContentLength,
	}
}

func (e *jsonLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	payload := map[string]interface{}{
		"time":          time.Now().Format(time.RFC3339),
		"id":            e.reqID,
		"remote_ip":     e.remoteIP,
		"host":          e.host,
		"method":        e.method,
		"uri":           e.uri,
		"status":        status,
		"error":         "",
		"latency":       elapsed.Nanoseconds(),
		"latency_human": elapsed.String(),
		"bytes_in":      e.bytesIn,
		"bytes_out":     bytes,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Println(string(data))
}

func (e *jsonLogEntry) Panic(value interface{}, stack []byte) {
	// Panics are handled and logged by chi_mw.Recoverer.
}

func createLoggerMiddleware() func(http.Handler) http.Handler {
	return chi_mw.RequestLogger(jsonLogFormatter{})
}

func createCORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-API-Key",
		},
		AllowCredentials: true,
		MaxAge:           86400,
	})
}

func createSecureMiddleware() func(http.Handler) http.Handler {
	csp := "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; " +
		"style-src 'self' 'unsafe-inline' https://cdnjs.cloudflare.com; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data: https://cdnjs.cloudflare.com; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none';"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-XSS-Protection", "1; mode=block")
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "SAMEORIGIN")
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubdomains; preload")
			header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			header.Set("Content-Security-Policy", csp)
			next.ServeHTTP(w, r)
		})
	}
}

func createConnection(address, serviceName string, logger logger.LoggerInterface, guard *resilience.DependencyGuardInterceptor) (*grpc.ClientConn, error) {
	logger.Info(fmt.Sprintf("Connecting to %s service at %s", serviceName, address))
	conn, err := server.NewGRPCClient(address, grpc.WithChainUnaryInterceptor(guard.UnaryInterceptor()))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s service", serviceName), zap.Error(err))
		return nil, err
	}
	return conn, nil
}

func closeConnections(conns handler.ServiceConnections, log logger.LoggerInterface) {
	connsMap := map[string]*grpc.ClientConn{
		"Auth":        conns.Auth,
		"Role":        conns.Role,
		"User":        conns.User,
		"Cashier":     conns.Cashier,
		"Category":    conns.Category,
		"Merchant":    conns.Merchant,
		"OrderItem":   conns.OrderItem,
		"Order":       conns.Order,
		"Product":     conns.Product,
		"Transaction": conns.Transaction,
		"StatsReader": conns.StatsReader,
	}

	for name, conn := range connsMap {
		if conn != nil {
			if err := conn.Close(); err != nil {
				log.Error(fmt.Sprintf("Failed to close %s connection", name), zap.Error(err))
			}
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
