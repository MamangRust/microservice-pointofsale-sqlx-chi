package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbmerchant "github.com/MamangRust/microservice-point-of-sale-pb/merchant"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type merchantHandleApi struct {
	client     pbmerchant.MerchantServiceClient
	logger     logger.LoggerInterface
	mapping    response_api.MerchantResponseMapper
	apiHandler apierror.ApiHandler
	cache      *gateway_cache.GatewayCache
}

func NewHandlerMerchant(
	router chi.Router,
	client pbmerchant.MerchantServiceClient,
	logger logger.LoggerInterface,
	mapping response_api.MerchantResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *merchantHandleApi {
	merchantHandler := &merchantHandleApi{
		client:     client,
		logger:     logger,
		mapping:    mapping,
		apiHandler: apiHandler,
		cache:      cache,
	}

	router.Route("/api/merchant", func(routerMerchant chi.Router) {

		routerMerchant.Get("/", apiHandler.Handle("get-merchant-findallmerchant", merchantHandler.FindAllMerchant))
		routerMerchant.Get("/{id}", apiHandler.Handle("get-merchant-findbyid", merchantHandler.FindById))
		routerMerchant.Get("/active", apiHandler.Handle("get-merchant-findbyactive", merchantHandler.FindByActive))
		routerMerchant.Get("/trashed", apiHandler.Handle("get-merchant-findbytrashed", merchantHandler.FindByTrashed))

		routerMerchant.Post("/create", apiHandler.Handle("post-merchant-create", merchantHandler.Create))
		routerMerchant.Post("/update/{id}", apiHandler.Handle("post-merchant-update", merchantHandler.Update))
		routerMerchant.Post("/update-status/{id}", apiHandler.Handle("post-merchant-updatestatus", merchantHandler.UpdateStatus))

		routerMerchant.Post("/trashed/{id}", apiHandler.Handle("post-merchant-trashedmerchant", merchantHandler.TrashedMerchant))
		routerMerchant.Post("/restore/{id}", apiHandler.Handle("post-merchant-restoremerchant", merchantHandler.RestoreMerchant))
		routerMerchant.Delete("/permanent/{id}", apiHandler.Handle("delete-merchant-deletemerchantpermanent", merchantHandler.DeleteMerchantPermanent))

		routerMerchant.Post("/restore/all", apiHandler.Handle("post-merchant-restoreallmerchant", merchantHandler.RestoreAllMerchant))
		routerMerchant.Post("/permanent/all", apiHandler.Handle("post-merchant-deleteallmerchantpermanent", merchantHandler.DeleteAllMerchantPermanent))

	})
	return merchantHandler
}

// List all merchant (paginated)
// List all merchant (paginated)
// List all merchant (paginated)
// @Summary List all merchant (paginated)
// @Tags Merchant
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchant
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant [get]
// @Security BearerAuth
func (h *merchantHandleApi) FindAllMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant:findallmerchant:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchant](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchant.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAll(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchant(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get merchant by ID
// Get merchant by ID
// Get merchant by ID
// @Summary Get merchant by ID
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchant
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/:id [get]
// @Security BearerAuth
func (h *merchantHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	cacheKey := fmt.Sprintf("merchant:findbyid:id_%d", id)
	if cached, found := gateway_cache.Get[response.ApiResponseMerchant](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchant.FindByIdMerchantRequest{
		Id: int32(id),
	}

	res, err := h.client.FindById(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchant(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List active merchant
// List active merchant
// List active merchant
// @Summary List active merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchantDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/active [get]
// @Security BearerAuth
func (h *merchantHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant:findbyactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchantDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchant.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchantDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List trashed merchant
// List trashed merchant
// List trashed merchant
// @Summary List trashed merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchantDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/trashed [get]
// @Security BearerAuth
func (h *merchantHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant:findbytrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchantDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchant.FindAllMerchantRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchantDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create merchant
// Create merchant
// Create merchant
// @Summary Create merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param request body requests.CreateMerchantRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchant
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/create [post]
// @Security BearerAuth
func (h *merchantHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var body requests.CreateMerchantRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchant.CreateMerchantRequest{
		UserId:       int32(body.UserID),
		Name:         body.Name,
		Description:  body.Description,
		Address:      body.Address,
		ContactEmail: body.ContactEmail,
		ContactPhone: body.ContactPhone,
		Status:       body.Status,
	}

	res, err := h.client.Create(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchant(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update merchant
// Update merchant
// Update merchant
// @Summary Update merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateMerchantRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchant
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/update/:id [post]
// @Security BearerAuth
func (h *merchantHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	var body requests.UpdateMerchantRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchant.UpdateMerchantRequest{
		MerchantId:   int32(id),
		UserId:       int32(body.UserID),
		Name:         body.Name,
		Description:  body.Description,
		Address:      body.Address,
		ContactEmail: body.ContactEmail,
		ContactPhone: body.ContactPhone,
		Status:       body.Status,
	}

	res, err := h.client.Update(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchant(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update merchant status
// Update merchant status
// Update merchant status
// @Summary Update merchant status
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateMerchantStatusRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchant
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/update-status/:id [post]
// @Security BearerAuth
func (h *merchantHandleApi) UpdateStatus(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	var body requests.UpdateMerchantStatusRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchant.UpdateMerchantStatusRequest{
		MerchantId: int32(id),
		Status:     body.Status,
	}

	res, err := h.client.UpdateMerchantStatus(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchant(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash merchant
// Trash merchant
// Trash merchant
// @Summary Trash merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/trashed/:id [post]
// @Security BearerAuth
func (h *merchantHandleApi) TrashedMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	req := &pbmerchant.FindByIdMerchantRequest{
		Id: int32(id),
	}

	res, err := h.client.TrashedMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore merchant
// Restore merchant
// Restore merchant
// @Summary Restore merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/restore/:id [post]
// @Security BearerAuth
func (h *merchantHandleApi) RestoreMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	req := &pbmerchant.FindByIdMerchantRequest{
		Id: int32(id),
	}

	res, err := h.client.RestoreMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchant(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete merchant permanently
// Delete merchant permanently
// Delete merchant permanently
// @Summary Delete merchant permanently
// @Tags Merchant
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDelete
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/permanent/:id [delete]
// @Security BearerAuth
func (h *merchantHandleApi) DeleteMerchantPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	req := &pbmerchant.FindByIdMerchantRequest{
		Id: int32(id),
	}

	res, err := h.client.DeleteMerchantPermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDelete(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed merchant
// Restore all trashed merchant
// Restore all trashed merchant
// @Summary Restore all trashed merchant
// @Tags Merchant
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseMerchantAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/restore/all [post]
// @Security BearerAuth
func (h *merchantHandleApi) RestoreAllMerchant(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.RestoreAllMerchant(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed merchant permanently
// Delete all trashed merchant permanently
// Delete all trashed merchant permanently
// @Summary Delete all trashed merchant permanently
// @Tags Merchant
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseMerchantAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant/permanent/all [post]
// @Security BearerAuth
func (h *merchantHandleApi) DeleteAllMerchantPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.DeleteAllMerchantPermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant:*")
	return httpx.JSON(w, http.StatusOK, so)
}
