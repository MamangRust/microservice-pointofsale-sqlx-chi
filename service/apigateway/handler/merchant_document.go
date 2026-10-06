package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbmerchantdoc "github.com/MamangRust/microservice-point-of-sale-pb/merchant_document"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type merchantDocumentHandleApi struct {
	merchantDocument pbmerchantdoc.MerchantDocumentServiceClient
	logger           logger.LoggerInterface
	mapping          response_api.MerchantDocumentResponseMapper
	apiHandler       apierror.ApiHandler
	cache            *gateway_cache.GatewayCache
}

func NewHandlerMerchantDocument(
	router chi.Router,
	merchantDocument pbmerchantdoc.MerchantDocumentServiceClient,
	logger logger.LoggerInterface,
	ma response_api.MerchantDocumentResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *merchantDocumentHandleApi {
	merchantDocumentHandler := &merchantDocumentHandleApi{
		merchantDocument: merchantDocument,
		logger:           logger,
		mapping:          ma,
		apiHandler:       apiHandler,
		cache:            cache,
	}

	router.Route("/api/merchant-documents", func(routerMerchantDocument chi.Router) {

		routerMerchantDocument.Get("/", apiHandler.Handle("get-merchantdocument-findall", merchantDocumentHandler.FindAll))
		routerMerchantDocument.Get("/{id}", apiHandler.Handle("get-merchantdocument-findbyid", merchantDocumentHandler.FindById))
		routerMerchantDocument.Get("/active", apiHandler.Handle("get-merchantdocument-findallactive", merchantDocumentHandler.FindAllActive))
		routerMerchantDocument.Get("/trashed", apiHandler.Handle("get-merchantdocument-findalltrashed", merchantDocumentHandler.FindAllTrashed))

		routerMerchantDocument.Post("/create", apiHandler.Handle("post-merchantdocument-create", merchantDocumentHandler.Create))
		routerMerchantDocument.Post("/updates/{id}", apiHandler.Handle("post-merchantdocument-update", merchantDocumentHandler.Update))
		routerMerchantDocument.Post("/update-status/{id}", apiHandler.Handle("post-merchantdocument-updatestatus", merchantDocumentHandler.UpdateStatus))

		routerMerchantDocument.Post("/trashed/{id}", apiHandler.Handle("post-merchantdocument-trasheddocument", merchantDocumentHandler.TrashedDocument))
		routerMerchantDocument.Post("/restore/{id}", apiHandler.Handle("post-merchantdocument-restoredocument", merchantDocumentHandler.RestoreDocument))
		routerMerchantDocument.Delete("/permanent/{id}", apiHandler.Handle("delete-merchantdocument-delete", merchantDocumentHandler.Delete))

		routerMerchantDocument.Post("/restore/all", apiHandler.Handle("post-merchantdocument-restorealldocuments", merchantDocumentHandler.RestoreAllDocuments))
		routerMerchantDocument.Post("/permanent/all", apiHandler.Handle("delete-merchantdocument-deletealldocumentspermanent", merchantDocumentHandler.DeleteAllDocumentsPermanent))

	})
	return merchantDocumentHandler
}

// List all merchant document (paginated)
// List all merchant document (paginated)
// List all merchant document (paginated)
// @Summary List all merchant document (paginated)
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents [get]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant_document:findall:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchantDocument](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchantdoc.FindAllMerchantDocumentsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.merchantDocument.FindAll(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchantDocument(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get merchant document by ID
// Get merchant document by ID
// Get merchant document by ID
// @Summary Get merchant document by ID
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/:id [get]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	cacheKey := fmt.Sprintf("merchant_document:findbyid:id_%d", id)
	if cached, found := gateway_cache.Get[response.ApiResponseMerchantDocument](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.merchantDocument.FindById(ctx, &pbmerchantdoc.FindMerchantDocumentByIdRequest{
		DocumentId: int32(id),
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List all merchant document (paginated)
// List all merchant document (paginated)
// List all merchant document (paginated)
// @Summary List all merchant document (paginated)
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/active [get]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) FindAllActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant_document:findallactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchantDocument](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchantdoc.FindAllMerchantDocumentsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.merchantDocument.FindAllActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchantDocument(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List all merchant document (paginated)
// List all merchant document (paginated)
// List all merchant document (paginated)
// @Summary List all merchant document (paginated)
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/trashed [get]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) FindAllTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("merchant_document:findalltrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationMerchantDocumentDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbmerchantdoc.FindAllMerchantDocumentsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.merchantDocument.FindAllTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationMerchantDocumentDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create merchant document
// Create merchant document
// Create merchant document
// @Summary Create merchant document
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param request body requests.CreateMerchantDocumentRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/create [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var body requests.CreateMerchantDocumentRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchantdoc.CreateMerchantDocumentRequest{
		MerchantId:   int32(body.MerchantID),
		DocumentType: body.DocumentType,
		DocumentUrl:  body.DocumentUrl,
	}

	res, err := h.merchantDocument.Create(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update merchant document
// Update merchant document
// Update merchant document
// @Summary Update merchant document
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateMerchantDocumentRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/updates/:id [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	var body requests.UpdateMerchantDocumentRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchantdoc.UpdateMerchantDocumentRequest{
		DocumentId:   int32(id),
		MerchantId:   int32(body.MerchantID),
		DocumentType: body.DocumentType,
		DocumentUrl:  body.DocumentUrl,
		Status:       body.Status,
		Note:         body.Note,
	}

	res, err := h.merchantDocument.Update(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update merchant document status
// Update merchant document status
// Update merchant document status
// @Summary Update merchant document status
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateMerchantDocumentStatusRequest true "Request body"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/update-status/:id [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) UpdateStatus(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	var body requests.UpdateMerchantDocumentStatusRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	req := &pbmerchantdoc.UpdateMerchantDocumentStatusRequest{
		DocumentId: int32(id),
		MerchantId: int32(body.MerchantID),
		Note:       body.Note,
		Status:     body.Status,
	}

	res, err := h.merchantDocument.UpdateStatus(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash merchant document
// Trash merchant document
// Trash merchant document
// @Summary Trash merchant document
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/trashed/:id [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) TrashedDocument(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	req := &pbmerchantdoc.TrashedMerchantDocumentRequest{DocumentId: int32(id)}
	res, err := h.merchantDocument.Trashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore merchant document
// Restore merchant document
// Restore merchant document
// @Summary Restore merchant document
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/restore/:id [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) RestoreDocument(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	req := &pbmerchantdoc.RestoreMerchantDocumentRequest{DocumentId: int32(id)}
	res, err := h.merchantDocument.Restore(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocument(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete
// Delete
// Delete
// @Summary Delete
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseMerchantDocument
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/permanent/:id [delete]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) Delete(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid document ID")
	}

	req := &pbmerchantdoc.DeleteMerchantDocumentPermanentRequest{DocumentId: int32(id)}
	res, err := h.merchantDocument.DeletePermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocumentDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed merchant document
// Restore all trashed merchant document
// Restore all trashed merchant document
// @Summary Restore all trashed merchant document
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseMerchantDocumentAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/restore/all [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) RestoreAllDocuments(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.merchantDocument.RestoreAll(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocumentAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed merchant document permanently
// Delete all trashed merchant document permanently
// Delete all trashed merchant document permanently
// @Summary Delete all trashed merchant document permanently
// @Tags Merchant Document
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseMerchantDocumentAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/merchant-documents/permanent/all [post]
// @Security BearerAuth
func (h *merchantDocumentHandleApi) DeleteAllDocumentsPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.merchantDocument.DeleteAllPermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseMerchantDocumentAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "merchant_document:*")
	return httpx.JSON(w, http.StatusOK, so)
}
