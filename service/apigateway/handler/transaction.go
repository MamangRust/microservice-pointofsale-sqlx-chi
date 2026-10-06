package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbtransaction "github.com/MamangRust/microservice-point-of-sale-pb/transaction"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type transactionHandleApi struct {
	client     pbtransaction.TransactionServiceClient
	logger     logger.LoggerInterface
	mapping    response_api.TransactionResponseMapper
	apiHandler apierror.ApiHandler
	cache      *gateway_cache.GatewayCache
}

func NewHandlerTransaction(
	router chi.Router,
	client pbtransaction.TransactionServiceClient,
	logger logger.LoggerInterface,
	mapping response_api.TransactionResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *transactionHandleApi {
	transactionHandle := &transactionHandleApi{
		client:     client,
		logger:     logger,
		mapping:    mapping,
		apiHandler: apiHandler,
		cache:      cache,
	}

	router.Route("/api/transaction", func(routerTransaction chi.Router) {

		routerTransaction.Get("/", apiHandler.Handle("get-transaction-findalltransaction", transactionHandle.FindAllTransaction))
		routerTransaction.Get("/{id}", apiHandler.Handle("get-transaction-findbyid", transactionHandle.FindById))
		routerTransaction.Get("/merchant/{merchant_id}", apiHandler.Handle("get-transaction-findbymerchant", transactionHandle.FindByMerchant))
		routerTransaction.Get("/active", apiHandler.Handle("get-transaction-findbyactive", transactionHandle.FindByActive))
		routerTransaction.Get("/trashed", apiHandler.Handle("get-transaction-findbytrashed", transactionHandle.FindByTrashed))

		routerTransaction.Post("/create", apiHandler.Handle("post-transaction-create", transactionHandle.Create))
		routerTransaction.Post("/update/{id}", apiHandler.Handle("post-transaction-update", transactionHandle.Update))

		routerTransaction.Post("/trashed/{id}", apiHandler.Handle("post-transaction-trashedtransaction", transactionHandle.TrashedTransaction))
		routerTransaction.Post("/restore/{id}", apiHandler.Handle("post-transaction-restoretransaction", transactionHandle.RestoreTransaction))
		routerTransaction.Delete("/permanent/{id}", apiHandler.Handle("delete-transaction-deletetransactionpermanent", transactionHandle.DeleteTransactionPermanent))

		routerTransaction.Post("/restore/all", apiHandler.Handle("post-transaction-restorealltransaction", transactionHandle.RestoreAllTransaction))
		routerTransaction.Post("/permanent/all", apiHandler.Handle("delete-transaction-deletealltransactionpermanent", transactionHandle.DeleteAllTransactionPermanent))

	})
	return transactionHandle
}

// List all transaction (paginated)
// List all transaction (paginated)
// List all transaction (paginated)
// @Summary List all transaction (paginated)
// @Tags Transaction
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationTransaction
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction [get]
// @Security BearerAuth
func (h *transactionHandleApi) FindAllTransaction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("transaction:findalltransaction:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationTransaction](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbtransaction.FindAllTransactionRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAll(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationTransaction(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List transaction by merchant
// List transaction by merchant
// List transaction by merchant
// @Summary List transaction by merchant
// @Tags Transaction
// @Accept json
// @Produce json
// @Param merchant_id path int true "merchant id"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationTransaction
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/merchant/:merchant_id [get]
// @Security BearerAuth
func (h *transactionHandleApi) FindByMerchant(w http.ResponseWriter, r *http.Request) error {
	merchantID, err := strconv.Atoi(chi.URLParam(r, "merchant_id"))
	if err != nil || merchantID <= 0 {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("transaction:findbymerchant:merchant_%d:page_%d:size_%d:search_%s", merchantID, page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationTransaction](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbtransaction.FindAllTransactionMerchantRequest{
		MerchantId: int32(merchantID),
		Page:       int32(page),
		PageSize:   int32(pageSize),
		Search:     search,
	}

	res, err := h.client.FindByMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationTransaction(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get transaction by ID
// Get transaction by ID
// Get transaction by ID
// @Summary Get transaction by ID
// @Tags Transaction
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseTransaction
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/:id [get]
// @Security BearerAuth
func (h *transactionHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid transaction ID")
	}

	cacheKey := fmt.Sprintf("transaction:findbyid:id_%d", id)
	if cached, found := gateway_cache.Get[response.ApiResponseTransaction](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbtransaction.FindByIdTransactionRequest{
		Id: int32(id),
	}

	res, err := h.client.FindById(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransaction(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List active transaction
// List active transaction
// List active transaction
// @Summary List active transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationTransactionDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/active [get]
// @Security BearerAuth
func (h *transactionHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("transaction:findbyactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationTransactionDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbtransaction.FindAllTransactionRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationTransactionDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List trashed transaction
// List trashed transaction
// List trashed transaction
// @Summary List trashed transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationTransactionDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/trashed [get]
// @Security BearerAuth
func (h *transactionHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("transaction:findbytrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationTransactionDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbtransaction.FindAllTransactionRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationTransactionDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create transaction
// Create transaction
// Create transaction
// @Summary Create transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param request body requests.CreateTransactionRequest true "Request body"
// @Success 200 {object} response.ApiResponseTransaction
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/create [post]
// @Security BearerAuth
func (h *transactionHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var body requests.CreateTransactionRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	grpcReq := &pbtransaction.CreateTransactionRequest{
		OrderId:       int32(body.OrderID),
		CashierId:     int32(body.CashierID),
		PaymentMethod: body.PaymentMethod,
		Amount:        int32(body.Amount),
	}

	res, err := h.client.Create(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransaction(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update transaction
// Update transaction
// Update transaction
// @Summary Update transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateTransactionRequest true "Request body"
// @Success 200 {object} response.ApiResponseTransaction
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/update/:id [post]
// @Security BearerAuth
func (h *transactionHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid transaction ID")
	}

	var body requests.UpdateTransactionRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	grpcReq := &pbtransaction.UpdateTransactionRequest{
		TransactionId: int32(id),
		OrderId:       int32(body.OrderID),
		CashierId:     int32(body.CashierID),
		PaymentMethod: body.PaymentMethod,
		Amount:        int32(body.Amount),
	}

	res, err := h.client.Update(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransaction(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash transaction
// Trash transaction
// Trash transaction
// @Summary Trash transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseTransactionDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/trashed/:id [post]
// @Security BearerAuth
func (h *transactionHandleApi) TrashedTransaction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid transaction ID")
	}

	req := &pbtransaction.FindByIdTransactionRequest{
		Id: int32(id),
	}

	res, err := h.client.TrashedTransaction(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore transaction
// Restore transaction
// Restore transaction
// @Summary Restore transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseTransactionDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/restore/:id [post]
// @Security BearerAuth
func (h *transactionHandleApi) RestoreTransaction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid transaction ID")
	}

	req := &pbtransaction.FindByIdTransactionRequest{
		Id: int32(id),
	}

	res, err := h.client.RestoreTransaction(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete transaction permanently
// Delete transaction permanently
// Delete transaction permanently
// @Summary Delete transaction permanently
// @Tags Transaction
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseTransactionDelete
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/permanent/:id [delete]
// @Security BearerAuth
func (h *transactionHandleApi) DeleteTransactionPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid transaction ID")
	}

	req := &pbtransaction.FindByIdTransactionRequest{
		Id: int32(id),
	}

	res, err := h.client.DeleteTransactionPermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionDelete(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed transaction
// Restore all trashed transaction
// Restore all trashed transaction
// @Summary Restore all trashed transaction
// @Tags Transaction
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseTransactionAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/restore/all [post]
// @Security BearerAuth
func (h *transactionHandleApi) RestoreAllTransaction(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.RestoreAllTransaction(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed transaction permanently
// Delete all trashed transaction permanently
// Delete all trashed transaction permanently
// @Summary Delete all trashed transaction permanently
// @Tags Transaction
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseTransactionAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/transaction/permanent/all [post]
// @Security BearerAuth
func (h *transactionHandleApi) DeleteAllTransactionPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.DeleteAllTransactionPermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseTransactionAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "transaction:*")
	return httpx.JSON(w, http.StatusOK, so)
}
