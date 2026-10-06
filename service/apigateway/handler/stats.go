package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbcashier "github.com/MamangRust/microservice-point-of-sale-pb/cashier"
	pbcategory "github.com/MamangRust/microservice-point-of-sale-pb/category"
	pborder "github.com/MamangRust/microservice-point-of-sale-pb/order"
	pbstats "github.com/MamangRust/microservice-point-of-sale-pb/stats"
	pbtransaction "github.com/MamangRust/microservice-point-of-sale-pb/transaction"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

// statsHandleApi exposes the stats-reader (ClickHouse) gRPC contracts as REST
// endpoints (F4 §7.4). It mirrors the full stats-reader surface: the 11 gRPC
// services / 50 rpc, i.e. order/cashier/category basic + ByMerchant + ById,
// plus transaction status and payment-method stats. All cache keys use the
// "apigw:" namespace so the gateway can never overwrite domain-service or
// stats-reader cache keys (§8.1).
type statsHandleApi struct {
	order              pbstats.OrderStatsServiceClient
	orderByMerchant    pbstats.OrderStatsByMerchantServiceClient
	orderById          pbstats.OrderStatsByIdServiceClient
	cashier            pbstats.CashierStatsServiceClient
	cashierByMerchant  pbstats.CashierStatsByMerchantServiceClient
	cashierById        pbstats.CashierStatsByIdServiceClient
	category           pbstats.CategoryStatsServiceClient
	categoryByMerchant pbstats.CategoryStatsByMerchantServiceClient
	categoryById       pbstats.CategoryStatsByIdServiceClient
	transaction        pbstats.TransactionStatsStatusServiceClient
	transactionMethod  pbstats.TransactionStatsMethodServiceClient
	logger             logger.LoggerInterface
	mapping            response_api.StatsResponseMapper
	apiHandler         apierror.ApiHandler
	cache              *gateway_cache.GatewayCache
}

// NewHandlerStats builds the 11 stats-reader clients from the shared
// StatsReader connection and registers every route.
func NewHandlerStats(
	router chi.Router,
	conn grpc.ClientConnInterface,
	logger logger.LoggerInterface,
	mapping response_api.StatsResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *statsHandleApi {
	statsHandler := &statsHandleApi{
		order:              pbstats.NewOrderStatsServiceClient(conn),
		orderByMerchant:    pbstats.NewOrderStatsByMerchantServiceClient(conn),
		orderById:          pbstats.NewOrderStatsByIdServiceClient(conn),
		cashier:            pbstats.NewCashierStatsServiceClient(conn),
		cashierByMerchant:  pbstats.NewCashierStatsByMerchantServiceClient(conn),
		cashierById:        pbstats.NewCashierStatsByIdServiceClient(conn),
		category:           pbstats.NewCategoryStatsServiceClient(conn),
		categoryByMerchant: pbstats.NewCategoryStatsByMerchantServiceClient(conn),
		categoryById:       pbstats.NewCategoryStatsByIdServiceClient(conn),
		transaction:        pbstats.NewTransactionStatsStatusServiceClient(conn),
		transactionMethod:  pbstats.NewTransactionStatsMethodServiceClient(conn),
		logger:             logger,
		mapping:            mapping,
		apiHandler:         apiHandler,
		cache:              cache,
	}

	router.Route("/api/order-stats", func(orderStats chi.Router) {
		orderStats.Get("/monthly-total-revenue", apiHandler.Handle("get-order-stats-monthly-total-revenue", statsHandler.FindMonthlyTotalRevenue))
		orderStats.Get("/yearly-total-revenue", apiHandler.Handle("get-order-stats-yearly-total-revenue", statsHandler.FindYearlyTotalRevenue))
		orderStats.Get("/monthly-revenue", apiHandler.Handle("get-order-stats-monthly-revenue", statsHandler.FindMonthlyRevenue))
		orderStats.Get("/yearly-revenue", apiHandler.Handle("get-order-stats-yearly-revenue", statsHandler.FindYearlyRevenue))

		orderStats.Route("/by-merchant", func(m chi.Router) {
			m.Get("/monthly-total-revenue", apiHandler.Handle("get-order-stats-monthly-total-revenue-by-merchant", statsHandler.FindMonthlyTotalRevenueByMerchant))
			m.Get("/yearly-total-revenue", apiHandler.Handle("get-order-stats-yearly-total-revenue-by-merchant", statsHandler.FindYearlyTotalRevenueByMerchant))
			m.Get("/monthly-revenue", apiHandler.Handle("get-order-stats-monthly-revenue-by-merchant", statsHandler.FindMonthlyRevenueByMerchant))
			m.Get("/yearly-revenue", apiHandler.Handle("get-order-stats-yearly-revenue-by-merchant", statsHandler.FindYearlyRevenueByMerchant))
		})

		orderStats.Route("/by-id", func(i chi.Router) {
			i.Get("/monthly-total-revenue", apiHandler.Handle("get-order-stats-monthly-total-revenue-by-id", statsHandler.FindMonthlyTotalRevenueById))
			i.Get("/yearly-total-revenue", apiHandler.Handle("get-order-stats-yearly-total-revenue-by-id", statsHandler.FindYearlyTotalRevenueById))
		})
	})

	router.Route("/api/cashier-stats", func(cashierStats chi.Router) {
		cashierStats.Get("/monthly-total-sales", apiHandler.Handle("get-cashier-stats-monthly-total-sales", statsHandler.FindCashierMonthlyTotalSales))
		cashierStats.Get("/yearly-total-sales", apiHandler.Handle("get-cashier-stats-yearly-total-sales", statsHandler.FindCashierYearlyTotalSales))
		cashierStats.Get("/month-sales", apiHandler.Handle("get-cashier-stats-month-sales", statsHandler.FindCashierMonthSales))
		cashierStats.Get("/year-sales", apiHandler.Handle("get-cashier-stats-year-sales", statsHandler.FindCashierYearSales))

		cashierStats.Route("/by-merchant", func(m chi.Router) {
			m.Get("/monthly-total-sales", apiHandler.Handle("get-cashier-stats-monthly-total-sales-by-merchant", statsHandler.FindCashierMonthlyTotalSalesByMerchant))
			m.Get("/yearly-total-sales", apiHandler.Handle("get-cashier-stats-yearly-total-sales-by-merchant", statsHandler.FindCashierYearlyTotalSalesByMerchant))
			m.Get("/month-sales", apiHandler.Handle("get-cashier-stats-month-sales-by-merchant", statsHandler.FindCashierMonthSalesByMerchant))
			m.Get("/year-sales", apiHandler.Handle("get-cashier-stats-year-sales-by-merchant", statsHandler.FindCashierYearSalesByMerchant))
		})

		cashierStats.Route("/by-id", func(i chi.Router) {
			i.Get("/monthly-total-sales", apiHandler.Handle("get-cashier-stats-monthly-total-sales-by-id", statsHandler.FindCashierMonthlyTotalSalesById))
			i.Get("/yearly-total-sales", apiHandler.Handle("get-cashier-stats-yearly-total-sales-by-id", statsHandler.FindCashierYearlyTotalSalesById))
			i.Get("/month-sales", apiHandler.Handle("get-cashier-stats-month-sales-by-id", statsHandler.FindCashierMonthSalesById))
			i.Get("/year-sales", apiHandler.Handle("get-cashier-stats-year-sales-by-id", statsHandler.FindCashierYearSalesById))
		})
	})

	router.Route("/api/category-stats", func(categoryStats chi.Router) {
		categoryStats.Get("/monthly-total-price", apiHandler.Handle("get-category-stats-monthly-total-price", statsHandler.FindCategoryMonthlyTotalPrice))
		categoryStats.Get("/yearly-total-price", apiHandler.Handle("get-category-stats-yearly-total-price", statsHandler.FindCategoryYearlyTotalPrice))
		categoryStats.Get("/month-price", apiHandler.Handle("get-category-stats-month-price", statsHandler.FindCategoryMonthPrice))
		categoryStats.Get("/year-price", apiHandler.Handle("get-category-stats-year-price", statsHandler.FindCategoryYearPrice))

		categoryStats.Route("/by-merchant", func(m chi.Router) {
			m.Get("/monthly-total-price", apiHandler.Handle("get-category-stats-monthly-total-price-by-merchant", statsHandler.FindCategoryMonthlyTotalPriceByMerchant))
			m.Get("/yearly-total-price", apiHandler.Handle("get-category-stats-yearly-total-price-by-merchant", statsHandler.FindCategoryYearlyTotalPriceByMerchant))
			m.Get("/month-price", apiHandler.Handle("get-category-stats-month-price-by-merchant", statsHandler.FindCategoryMonthPriceByMerchant))
			m.Get("/year-price", apiHandler.Handle("get-category-stats-year-price-by-merchant", statsHandler.FindCategoryYearPriceByMerchant))
		})

		categoryStats.Route("/by-id", func(i chi.Router) {
			i.Get("/monthly-total-price", apiHandler.Handle("get-category-stats-monthly-total-price-by-id", statsHandler.FindCategoryMonthlyTotalPriceById))
			i.Get("/yearly-total-price", apiHandler.Handle("get-category-stats-yearly-total-price-by-id", statsHandler.FindCategoryYearlyTotalPriceById))
			i.Get("/month-price", apiHandler.Handle("get-category-stats-month-price-by-id", statsHandler.FindCategoryMonthPriceById))
			i.Get("/year-price", apiHandler.Handle("get-category-stats-year-price-by-id", statsHandler.FindCategoryYearPriceById))
		})
	})

	router.Route("/api/transaction-stats", func(transactionStats chi.Router) {
		transactionStats.Get("/monthly-success", apiHandler.Handle("get-transaction-stats-monthly-success", statsHandler.FindTransactionMonthlySuccess))
		transactionStats.Get("/yearly-success", apiHandler.Handle("get-transaction-stats-yearly-success", statsHandler.FindTransactionYearlySuccess))
		transactionStats.Get("/monthly-failed", apiHandler.Handle("get-transaction-stats-monthly-failed", statsHandler.FindTransactionMonthlyFailed))
		transactionStats.Get("/yearly-failed", apiHandler.Handle("get-transaction-stats-yearly-failed", statsHandler.FindTransactionYearlyFailed))

		transactionStats.Route("/by-merchant", func(m chi.Router) {
			m.Get("/monthly-success", apiHandler.Handle("get-transaction-stats-monthly-success-by-merchant", statsHandler.FindTransactionMonthlySuccessByMerchant))
			m.Get("/yearly-success", apiHandler.Handle("get-transaction-stats-yearly-success-by-merchant", statsHandler.FindTransactionYearlySuccessByMerchant))
			m.Get("/monthly-failed", apiHandler.Handle("get-transaction-stats-monthly-failed-by-merchant", statsHandler.FindTransactionMonthlyFailedByMerchant))
			m.Get("/yearly-failed", apiHandler.Handle("get-transaction-stats-yearly-failed-by-merchant", statsHandler.FindTransactionYearlyFailedByMerchant))
		})

		transactionStats.Route("/method", func(method chi.Router) {
			method.Get("/monthly-success", apiHandler.Handle("get-transaction-method-monthly-success", statsHandler.FindTransactionMonthMethodSuccess))
			method.Get("/yearly-success", apiHandler.Handle("get-transaction-method-yearly-success", statsHandler.FindTransactionYearMethodSuccess))
			method.Get("/monthly-failed", apiHandler.Handle("get-transaction-method-monthly-failed", statsHandler.FindTransactionMonthMethodFailed))
			method.Get("/yearly-failed", apiHandler.Handle("get-transaction-method-yearly-failed", statsHandler.FindTransactionYearMethodFailed))

			method.Route("/by-merchant", func(m chi.Router) {
				m.Get("/monthly-success", apiHandler.Handle("get-transaction-method-monthly-success-by-merchant", statsHandler.FindTransactionMonthMethodByMerchantSuccess))
				m.Get("/yearly-success", apiHandler.Handle("get-transaction-method-yearly-success-by-merchant", statsHandler.FindTransactionYearMethodByMerchantSuccess))
				m.Get("/monthly-failed", apiHandler.Handle("get-transaction-method-monthly-failed-by-merchant", statsHandler.FindTransactionMonthMethodByMerchantFailed))
				m.Get("/yearly-failed", apiHandler.Handle("get-transaction-method-yearly-failed-by-merchant", statsHandler.FindTransactionYearMethodByMerchantFailed))
			})
		})
	})

	return statsHandler
}

func (h *statsHandleApi) parseYearMonth(r *http.Request) (int, int, error) {
	year, err := parseQueryIntWithValidation(r, "year", 1, 9999)
	if err != nil {
		return 0, 0, errors.NewBadRequestError("invalid year")
	}
	month, err := parseQueryIntWithValidation(r, "month", 1, 12)
	if err != nil {
		return 0, 0, errors.NewBadRequestError("invalid month")
	}
	return year, month, nil
}

func (h *statsHandleApi) parseYear(r *http.Request) (int, error) {
	year, err := parseQueryIntWithValidation(r, "year", 1, 9999)
	if err != nil {
		return 0, errors.NewBadRequestError("invalid year")
	}
	return year, nil
}

// parseID validates a positive int32 id query param (merchant_id, order_id,
// cashier_id, category_id).
func (h *statsHandleApi) parseID(r *http.Request, key string) (int, error) {
	id, err := parseQueryIntWithValidation(r, key, 1, 2147483647)
	if err != nil {
		return 0, errors.NewBadRequestError("invalid " + key)
	}
	return id, nil
}

// ── Order stats ─────────────────────────────────────────────────────────────

// @Summary Monthly total revenue (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponseOrderMonthlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/monthly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindMonthlyTotalRevenue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:monthly-total-revenue:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderMonthlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.order.FindMonthlyTotalRevenue(ctx, &pborder.FindYearMonthTotalRevenue{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderMonthlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total revenue (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseOrderYearlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/yearly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindYearlyTotalRevenue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:yearly-total-revenue:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderYearlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.order.FindYearlyTotalRevenue(ctx, &pborder.FindYearTotalRevenue{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderYearlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly revenue (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseOrderMonthly
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/monthly-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindMonthlyRevenue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:monthly-revenue:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderMonthly](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.order.FindMonthlyRevenue(ctx, &pborder.FindYearOrder{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderMonthly(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly revenue (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseOrderYearly
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/yearly-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindYearlyRevenue(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:yearly-revenue:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderYearly](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.order.FindYearlyRevenue(ctx, &pborder.FindYearOrder{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderYearly(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total revenue by merchant (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseOrderMonthlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-merchant/monthly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindMonthlyTotalRevenueByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:monthly-total-revenue:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderMonthlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderByMerchant.FindMonthlyTotalRevenueByMerchant(ctx, &pborder.FindYearMonthTotalRevenueByMerchant{Year: int32(year), Month: int32(month), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderMonthlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total revenue by merchant (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseOrderYearlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-merchant/yearly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindYearlyTotalRevenueByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:yearly-total-revenue:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderYearlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderByMerchant.FindYearlyTotalRevenueByMerchant(ctx, &pborder.FindYearTotalRevenueByMerchant{Year: int32(year), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderYearlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly revenue by merchant (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseOrderMonthly
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-merchant/monthly-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindMonthlyRevenueByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:monthly-revenue:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderMonthly](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderByMerchant.FindMonthlyRevenueByMerchant(ctx, &pborder.FindYearOrderByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderMonthly(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly revenue by merchant (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseOrderYearly
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-merchant/yearly-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindYearlyRevenueByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:yearly-revenue:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderYearly](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderByMerchant.FindYearlyRevenueByMerchant(ctx, &pborder.FindYearOrderByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderYearly(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total revenue by order (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param order_id query int true "Order ID"
// @Success 200 {object} response.ApiResponseOrderMonthlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-id/monthly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindMonthlyTotalRevenueById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	orderID, err := h.parseID(r, "order_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:monthly-total-revenue:id:%d:%d:%d", orderID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderMonthlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderById.FindMonthlyTotalRevenueById(ctx, &pborder.FindYearMonthTotalRevenueById{Year: int32(year), Month: int32(month), OrderId: int32(orderID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderMonthlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total revenue by order (order)
// @Tags Order Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param order_id query int true "Order ID"
// @Success 200 {object} response.ApiResponseOrderYearlyTotalRevenue
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order-stats/by-id/yearly-total-revenue [get]
// @Security BearerAuth
func (h *statsHandleApi) FindYearlyTotalRevenueById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	orderID, err := h.parseID(r, "order_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:order-stats:yearly-total-revenue:id:%d:%d", orderID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseOrderYearlyTotalRevenue](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.orderById.FindYearlyTotalRevenueById(ctx, &pborder.FindYearTotalRevenueById{Year: int32(year), OrderId: int32(orderID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderYearlyTotalRevenue(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// ── Cashier stats ───────────────────────────────────────────────────────────

// @Summary Monthly total sales (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponseCashierMonthlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/monthly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthlyTotalSales(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:monthly-total-sales:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashier.FindMonthlyTotalSales(ctx, &pbcashier.FindYearMonthTotalSales{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total sales (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCashierYearlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/yearly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearlyTotalSales(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:yearly-total-sales:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashier.FindYearlyTotalSales(ctx, &pbcashier.FindYearTotalSales{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month sales (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCashierMonthSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/month-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthSales(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:month-sales:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashier.FindMonthSales(ctx, &pbcashier.FindYearCashier{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year sales (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCashierYearSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/year-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearSales(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:year-sales:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashier.FindYearSales(ctx, &pbcashier.FindYearCashier{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total sales by merchant (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCashierMonthlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-merchant/monthly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthlyTotalSalesByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:monthly-total-sales:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierByMerchant.FindMonthlyTotalSalesByMerchant(ctx, &pbcashier.FindYearMonthTotalSalesByMerchant{Year: int32(year), Month: int32(month), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total sales by merchant (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCashierYearlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-merchant/yearly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearlyTotalSalesByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:yearly-total-sales:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierByMerchant.FindYearlyTotalSalesByMerchant(ctx, &pbcashier.FindYearTotalSalesByMerchant{Year: int32(year), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month sales by merchant (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCashierMonthSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-merchant/month-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthSalesByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:month-sales:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierByMerchant.FindMonthSalesByMerchant(ctx, &pbcashier.FindYearCashierByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year sales by merchant (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCashierYearSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-merchant/year-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearSalesByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:year-sales:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierByMerchant.FindYearSalesByMerchant(ctx, &pbcashier.FindYearCashierByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total sales by cashier (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param cashier_id query int true "Cashier ID"
// @Success 200 {object} response.ApiResponseCashierMonthlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-id/monthly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthlyTotalSalesById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	cashierID, err := h.parseID(r, "cashier_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:monthly-total-sales:id:%d:%d:%d", cashierID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierById.FindMonthlyTotalSalesById(ctx, &pbcashier.FindYearMonthTotalSalesById{Year: int32(year), Month: int32(month), CashierId: int32(cashierID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total sales by cashier (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param cashier_id query int true "Cashier ID"
// @Success 200 {object} response.ApiResponseCashierYearlyTotalSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-id/yearly-total-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearlyTotalSalesById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	cashierID, err := h.parseID(r, "cashier_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:yearly-total-sales:id:%d:%d", cashierID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearlyTotalSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierById.FindYearlyTotalSalesById(ctx, &pbcashier.FindYearTotalSalesById{Year: int32(year), CashierId: int32(cashierID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearlyTotalSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month sales by cashier (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param cashier_id query int true "Cashier ID"
// @Success 200 {object} response.ApiResponseCashierMonthSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-id/month-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierMonthSalesById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	cashierID, err := h.parseID(r, "cashier_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:month-sales:id:%d:%d", cashierID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierMonthSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierById.FindMonthSalesById(ctx, &pbcashier.FindYearCashierById{CashierId: int32(cashierID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierMonthSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year sales by cashier (cashier)
// @Tags Cashier Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param cashier_id query int true "Cashier ID"
// @Success 200 {object} response.ApiResponseCashierYearSales
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/cashier-stats/by-id/year-sales [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCashierYearSalesById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	cashierID, err := h.parseID(r, "cashier_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:cashier-stats:year-sales:id:%d:%d", cashierID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCashierYearSales](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.cashierById.FindYearSalesById(ctx, &pbcashier.FindYearCashierById{CashierId: int32(cashierID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCashierYearSales(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// ── Category stats ──────────────────────────────────────────────────────────

// @Summary Monthly total price (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponseCategoryMonthlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/monthly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthlyTotalPrice(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:monthly-total-price:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.category.FindMonthlyTotalPrices(ctx, &pbcategory.FindYearMonthTotalPrices{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total price (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCategoryYearlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/yearly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearlyTotalPrice(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:yearly-total-price:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.category.FindYearlyTotalPrices(ctx, &pbcategory.FindYearTotalPrices{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month price (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCategoryMonthPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/month-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthPrice(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:month-price:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.category.FindMonthPrice(ctx, &pbcategory.FindYearCategory{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year price (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponseCategoryYearPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/year-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearPrice(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:year-price:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.category.FindYearPrice(ctx, &pbcategory.FindYearCategory{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total price by merchant (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCategoryMonthlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-merchant/monthly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthlyTotalPriceByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:monthly-total-price:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryByMerchant.FindMonthlyTotalPricesByMerchant(ctx, &pbcategory.FindYearMonthTotalPriceByMerchant{Year: int32(year), Month: int32(month), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total price by merchant (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCategoryYearlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-merchant/yearly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearlyTotalPriceByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:yearly-total-price:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryByMerchant.FindYearlyTotalPricesByMerchant(ctx, &pbcategory.FindYearTotalPriceByMerchant{Year: int32(year), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month price by merchant (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCategoryMonthPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-merchant/month-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthPriceByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:month-price:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryByMerchant.FindMonthPriceByMerchant(ctx, &pbcategory.FindYearCategoryByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year price by merchant (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponseCategoryYearPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-merchant/year-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearPriceByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:year-price:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryByMerchant.FindYearPriceByMerchant(ctx, &pbcategory.FindYearCategoryByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly total price by category (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param category_id query int true "Category ID"
// @Success 200 {object} response.ApiResponseCategoryMonthlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-id/monthly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthlyTotalPriceById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	categoryID, err := h.parseID(r, "category_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:monthly-total-price:id:%d:%d:%d", categoryID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryById.FindMonthlyTotalPricesById(ctx, &pbcategory.FindYearMonthTotalPriceById{Year: int32(year), Month: int32(month), CategoryId: int32(categoryID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly total price by category (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param category_id query int true "Category ID"
// @Success 200 {object} response.ApiResponseCategoryYearlyTotalPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-id/yearly-total-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearlyTotalPriceById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	categoryID, err := h.parseID(r, "category_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:yearly-total-price:id:%d:%d", categoryID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearlyTotalPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryById.FindYearlyTotalPricesById(ctx, &pbcategory.FindYearTotalPriceById{Year: int32(year), CategoryId: int32(categoryID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearlyTotalPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Month price by category (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param category_id query int true "Category ID"
// @Success 200 {object} response.ApiResponseCategoryMonthPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-id/month-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryMonthPriceById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	categoryID, err := h.parseID(r, "category_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:month-price:id:%d:%d", categoryID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryMonthPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryById.FindMonthPriceById(ctx, &pbcategory.FindYearCategoryById{CategoryId: int32(categoryID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryMonthPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Year price by category (category)
// @Tags Category Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param category_id query int true "Category ID"
// @Success 200 {object} response.ApiResponseCategoryYearPrice
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/category-stats/by-id/year-price [get]
// @Security BearerAuth
func (h *statsHandleApi) FindCategoryYearPriceById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	categoryID, err := h.parseID(r, "category_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:category-stats:year-price:id:%d:%d", categoryID, year)
	if cached, found := gateway_cache.Get[response.ApiResponseCategoryYearPrice](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.categoryById.FindYearPriceById(ctx, &pbcategory.FindYearCategoryById{CategoryId: int32(categoryID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseCategoryYearPrice(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// ── Transaction status stats ────────────────────────────────────────────────

// @Summary Monthly success (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponsesTransactionMonthSuccess
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/monthly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthlySuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:monthly-success:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthSuccess](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindMonthStatusSuccess(ctx, &pbtransaction.FindMonthlyTransactionStatus{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthSuccess(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly success (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponsesTransactionYearSuccess
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/yearly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearlySuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:yearly-success:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearSuccess](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindYearStatusSuccess(ctx, &pbtransaction.FindYearlyTransactionStatus{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearSuccess(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly failed (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponsesTransactionMonthFailed
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/monthly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthlyFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:monthly-failed:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthFailed](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindMonthStatusFailed(ctx, &pbtransaction.FindMonthlyTransactionStatus{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthFailed(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly failed (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponsesTransactionYearFailed
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/yearly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearlyFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:yearly-failed:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearFailed](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindYearStatusFailed(ctx, &pbtransaction.FindYearlyTransactionStatus{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearFailed(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly success by merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionMonthSuccess
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/by-merchant/monthly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthlySuccessByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:monthly-success:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthSuccess](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindMonthStatusSuccessByMerchant(ctx, &pbtransaction.FindMonthlyTransactionStatusByMerchant{Year: int32(year), Month: int32(month), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthSuccess(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly success by merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionYearSuccess
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/by-merchant/yearly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearlySuccessByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:yearly-success:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearSuccess](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindYearStatusSuccessByMerchant(ctx, &pbtransaction.FindYearlyTransactionStatusByMerchant{Year: int32(year), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearSuccess(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly failed by merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionMonthFailed
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/by-merchant/monthly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthlyFailedByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:monthly-failed:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthFailed](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindMonthStatusFailedByMerchant(ctx, &pbtransaction.FindMonthlyTransactionStatusByMerchant{Year: int32(year), Month: int32(month), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthFailed(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly failed by merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionYearFailed
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/by-merchant/yearly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearlyFailedByMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-stats:yearly-failed:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearFailed](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transaction.FindYearStatusFailedByMerchant(ctx, &pbtransaction.FindYearlyTransactionStatusByMerchant{Year: int32(year), MerchantId: int32(merchantID)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearFailed(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// ── Transaction payment-method stats ────────────────────────────────────────

// @Summary Monthly success by payment method (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponsesTransactionMonthMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/monthly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthMethodSuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:monthly-success:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindMonthMethodSuccess(ctx, &pbtransaction.MonthTransactionMethod{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly success by payment method (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponsesTransactionYearMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/yearly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearMethodSuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:yearly-success:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindYearMethodSuccess(ctx, &pbtransaction.YearTransactionMethod{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly failed by payment method (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Success 200 {object} response.ApiResponsesTransactionMonthMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/monthly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthMethodFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:monthly-failed:%d:%d", year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindMonthMethodFailed(ctx, &pbtransaction.MonthTransactionMethod{Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly failed by payment method (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Success 200 {object} response.ApiResponsesTransactionYearMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/yearly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearMethodFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:yearly-failed:%d", year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindYearMethodFailed(ctx, &pbtransaction.YearTransactionMethod{Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly success by payment method and merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionMonthMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/by-merchant/monthly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthMethodByMerchantSuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:monthly-success:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindMonthMethodByMerchantSuccess(ctx, &pbtransaction.MonthTransactionMethodByMerchant{MerchantId: int32(merchantID), Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly success by payment method and merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionYearMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/by-merchant/yearly-success [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearMethodByMerchantSuccess(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:yearly-success:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindYearMethodByMerchantSuccess(ctx, &pbtransaction.YearTransactionMethodByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Monthly failed by payment method and merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param month query int true "Month (1-12)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionMonthMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/by-merchant/monthly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionMonthMethodByMerchantFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, month, err := h.parseYearMonth(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:monthly-failed:merchant:%d:%d:%d", merchantID, year, month)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionMonthMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindMonthMethodByMerchantFailed(ctx, &pbtransaction.MonthTransactionMethodByMerchant{MerchantId: int32(merchantID), Year: int32(year), Month: int32(month)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionMonthMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// @Summary Yearly failed by payment method and merchant (transaction)
// @Tags Transaction Stats
// @Accept json
// @Produce json
// @Param year query int true "Year (e.g. 2026)"
// @Param merchant_id query int true "Merchant ID"
// @Success 200 {object} response.ApiResponsesTransactionYearMethod
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction-stats/method/by-merchant/yearly-failed [get]
// @Security BearerAuth
func (h *statsHandleApi) FindTransactionYearMethodByMerchantFailed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	year, err := h.parseYear(r)
	if err != nil {
		return err
	}
	merchantID, err := h.parseID(r, "merchant_id")
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("apigw:transaction-method-stats:yearly-failed:merchant:%d:%d", merchantID, year)
	if cached, found := gateway_cache.Get[response.ApiResponsesTransactionYearMethod](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.transactionMethod.FindYearMethodByMerchantFailed(ctx, &pbtransaction.YearTransactionMethodByMerchant{MerchantId: int32(merchantID), Year: int32(year)})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionYearMethod(res)
	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}
