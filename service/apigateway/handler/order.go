package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pborder "github.com/MamangRust/microservice-point-of-sale-pb/order"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type orderHandleApi struct {
	client     pborder.OrderServiceClient
	logger     logger.LoggerInterface
	mapping    response_api.OrderResponseMapper
	apiHandler apierror.ApiHandler
	cache      *gateway_cache.GatewayCache
}

func NewHandlerOrder(
	router chi.Router,
	client pborder.OrderServiceClient,
	logger logger.LoggerInterface,
	mapping response_api.OrderResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *orderHandleApi {
	orderHandler := &orderHandleApi{
		client:     client,
		logger:     logger,
		mapping:    mapping,
		apiHandler: apiHandler,
		cache:      cache,
	}

	router.Route("/api/order", func(routerOrder chi.Router) {

		routerOrder.Get("/", apiHandler.Handle("get-order-findallorders", orderHandler.FindAllOrders))
		routerOrder.Get("/{id}", apiHandler.Handle("get-order-findbyid", orderHandler.FindById))
		routerOrder.Get("/active", apiHandler.Handle("get-order-findbyactive", orderHandler.FindByActive))
		routerOrder.Get("/trashed", apiHandler.Handle("get-order-findbytrashed", orderHandler.FindByTrashed))

		routerOrder.Post("/create", apiHandler.Handle("post-order-create", orderHandler.Create))
		routerOrder.Post("/update/{id}", apiHandler.Handle("post-order-update", orderHandler.Update))

		routerOrder.Post("/trashed/{id}", apiHandler.Handle("post-order-trashedorder", orderHandler.TrashedOrder))
		routerOrder.Post("/restore/{id}", apiHandler.Handle("post-order-restoreorder", orderHandler.RestoreOrder))
		routerOrder.Delete("/permanent/{id}", apiHandler.Handle("delete-order-deleteorderpermanent", orderHandler.DeleteOrderPermanent))

		routerOrder.Post("/restore/all", apiHandler.Handle("post-order-restoreallorder", orderHandler.RestoreAllOrder))
		routerOrder.Post("/permanent/all", apiHandler.Handle("delete-order-deleteallorderpermanent", orderHandler.DeleteAllOrderPermanent))

	})
	return orderHandler
}

// List all order (paginated)
// List all order (paginated)
// List all order (paginated)
// @Summary List all order (paginated)
// @Tags Order
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationOrder
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order [get]
// @Security BearerAuth
func (h *orderHandleApi) FindAllOrders(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("order:findallorders:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationOrder](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pborder.FindAllOrderRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAll(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationOrder(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get order by ID
// Get order by ID
// Get order by ID
// @Summary Get order by ID
// @Tags Order
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseOrder
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/:id [get]
// @Security BearerAuth
func (h *orderHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid order ID")
	}

	cacheKey := fmt.Sprintf("order:findbyid:id_%d", id)
	if cached, found := gateway_cache.Get[response.ApiResponseOrder](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pborder.FindByIdOrderRequest{
		Id: int32(id),
	}

	res, err := h.client.FindById(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrder(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List active order
// List active order
// List active order
// @Summary List active order
// @Tags Order
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationOrderDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/active [get]
// @Security BearerAuth
func (h *orderHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("order:findbyactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationOrderDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pborder.FindAllOrderRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationOrderDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List trashed order
// List trashed order
// List trashed order
// @Summary List trashed order
// @Tags Order
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationOrderDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/trashed [get]
// @Security BearerAuth
func (h *orderHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("order:findbytrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationOrderDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pborder.FindAllOrderRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationOrderDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create order
// Create order
// Create order
// @Summary Create order
// @Tags Order
// @Accept json
// @Produce json
// @Param request body requests.CreateOrderRequest true "Request body"
// @Success 200 {object} response.ApiResponseOrder
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/create [post]
// @Security BearerAuth
func (h *orderHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var body requests.CreateOrderRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	grpcReq := &pborder.CreateOrderRequest{
		MerchantId: int32(body.MerchantID),
		CashierId:  int32(body.CashierID),
	}

	for _, item := range body.Items {
		grpcReq.Items = append(grpcReq.Items, &pborder.CreateOrderItemRequest{
			ProductId: int32(item.ProductID),
			Quantity:  int32(item.Quantity),
		})
	}

	res, err := h.client.Create(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrder(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update order
// Update order
// Update order
// @Summary Update order
// @Tags Order
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateOrderRequest true "Request body"
// @Success 200 {object} response.ApiResponseOrder
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/update/:id [post]
// @Security BearerAuth
func (h *orderHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid order ID")
	}

	var body requests.UpdateOrderRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	grpcReq := &pborder.UpdateOrderRequest{
		OrderId: int32(id),
		Items:   []*pborder.UpdateOrderItemRequest{},
	}

	for _, item := range body.Items {
		grpcReq.Items = append(grpcReq.Items, &pborder.UpdateOrderItemRequest{
			OrderItemId: int32(item.OrderItemID),
			ProductId:   int32(item.ProductID),
			Quantity:    int32(item.Quantity),
		})
	}

	res, err := h.client.Update(ctx, grpcReq)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrder(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash order
// Trash order
// Trash order
// @Summary Trash order
// @Tags Order
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseOrderDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/trashed/:id [post]
// @Security BearerAuth
func (h *orderHandleApi) TrashedOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid order ID")
	}

	req := &pborder.FindByIdOrderRequest{
		Id: int32(id),
	}

	res, err := h.client.TrashedOrder(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore order
// Restore order
// Restore order
// @Summary Restore order
// @Tags Order
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseOrderDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/restore/:id [post]
// @Security BearerAuth
func (h *orderHandleApi) RestoreOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid order ID")
	}

	req := &pborder.FindByIdOrderRequest{
		Id: int32(id),
	}

	res, err := h.client.RestoreOrder(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete order permanently
// Delete order permanently
// Delete order permanently
// @Summary Delete order permanently
// @Tags Order
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseOrderDelete
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/permanent/:id [delete]
// @Security BearerAuth
func (h *orderHandleApi) DeleteOrderPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid order ID")
	}

	req := &pborder.FindByIdOrderRequest{
		Id: int32(id),
	}

	res, err := h.client.DeleteOrderPermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderDelete(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed order
// Restore all trashed order
// Restore all trashed order
// @Summary Restore all trashed order
// @Tags Order
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseOrderAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/restore/all [post]
// @Security BearerAuth
func (h *orderHandleApi) RestoreAllOrder(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.RestoreAllOrder(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed order permanently
// Delete all trashed order permanently
// Delete all trashed order permanently
// @Summary Delete all trashed order permanently
// @Tags Order
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseOrderAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/order/permanent/all [post]
// @Security BearerAuth
func (h *orderHandleApi) DeleteAllOrderPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.DeleteAllOrderPermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseOrderAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "order:*")
	return httpx.JSON(w, http.StatusOK, so)
}
