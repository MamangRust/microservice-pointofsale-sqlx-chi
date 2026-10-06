package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbproduct "github.com/MamangRust/microservice-point-of-sale-pb/product"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-pkg/upload_image"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type productHandleApi struct {
	client       pbproduct.ProductServiceClient
	logger       logger.LoggerInterface
	mapping      response_api.ProductResponseMapper
	upload_image upload_image.ImageUploads
	apiHandler   apierror.ApiHandler
	cache        *gateway_cache.GatewayCache
}

func NewHandlerProduct(
	router chi.Router,
	client pbproduct.ProductServiceClient,
	logger logger.LoggerInterface,
	mapping response_api.ProductResponseMapper,
	upload_image upload_image.ImageUploads,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *productHandleApi {
	productHandler := &productHandleApi{
		client:       client,
		logger:       logger,
		mapping:      mapping,
		upload_image: upload_image,
		apiHandler:   apiHandler,
		cache:        cache,
	}

	router.Route("/api/product", func(routerProduct chi.Router) {
		routerProduct.Get("/", apiHandler.Handle("get-product-findallproduct", productHandler.FindAllProduct))
		routerProduct.Get("/{id}", apiHandler.Handle("get-product-findbyid", productHandler.FindById))
		routerProduct.Get("/merchant/{merchant_id}", apiHandler.Handle("get-product-findbymerchant", productHandler.FindByMerchant))
		routerProduct.Get("/category/{category_name}", apiHandler.Handle("get-product-findbycategory", productHandler.FindByCategory))

		routerProduct.Get("/active", apiHandler.Handle("get-product-findbyactive", productHandler.FindByActive))
		routerProduct.Get("/trashed", apiHandler.Handle("get-product-findbytrashed", productHandler.FindByTrashed))

		routerProduct.Post("/create", apiHandler.Handle("post-product-create", productHandler.Create))
		routerProduct.Post("/update/{id}", apiHandler.Handle("post-product-update", productHandler.Update))

		routerProduct.Post("/trashed/{id}", apiHandler.Handle("post-product-trashedproduct", productHandler.TrashedProduct))
		routerProduct.Post("/restore/{id}", apiHandler.Handle("post-product-restoreproduct", productHandler.RestoreProduct))
		routerProduct.Delete("/permanent/{id}", apiHandler.Handle("delete-product-deleteproductpermanent", productHandler.DeleteProductPermanent))

		routerProduct.Post("/restore/all", apiHandler.Handle("post-product-restoreallproduct", productHandler.RestoreAllProduct))
		routerProduct.Post("/permanent/all", apiHandler.Handle("delete-product-deleteallproductpermanent", productHandler.DeleteAllProductPermanent))

	})
	return productHandler
}

// List all product (paginated)
// List all product (paginated)
// List all product (paginated)
// @Summary List all product (paginated)
// @Tags Product
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product [get]
// @Security BearerAuth
func (h *productHandleApi) FindAllProduct(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("product:findallproduct:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationProduct](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindAllProductRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAll(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationProduct(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List product by merchant
// List product by merchant
// List product by merchant
// @Summary List product by merchant
// @Tags Product
// @Accept json
// @Produce json
// @Param merchant_id path int true "merchant id"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsesProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/merchant/:merchant_id [get]
// @Security BearerAuth
func (h *productHandleApi) FindByMerchant(w http.ResponseWriter, r *http.Request) error {
	merchantID, err := strconv.Atoi(chi.URLParam(r, "merchant_id"))
	if err != nil || merchantID <= 0 {
		return errors.NewBadRequestError("invalid merchant ID")
	}

	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	categoryIDStr := r.URL.Query().Get("category_id")
	minPriceStr := r.URL.Query().Get("min_price")
	maxPriceStr := r.URL.Query().Get("max_price")

	cacheKey := fmt.Sprintf("product:findbymerchant:merchant_%d:page_%d:size_%d:search_%s:cat_%s:min_%s:max_%s",
		merchantID, page, pageSize, search, categoryIDStr, minPriceStr, maxPriceStr)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationProduct](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindAllProductMerchantRequest{
		MerchantId: int32(merchantID),
		Page:       int32(page),
		PageSize:   int32(pageSize),
		Search:     search,
	}

	if categoryIDStr != "" {
		if id, err := strconv.Atoi(categoryIDStr); err == nil && id > 0 {
			req.CategoryId = int32(id)
		}
	}

	if minPriceStr != "" {
		if price, err := strconv.Atoi(minPriceStr); err == nil && price >= 0 {
			req.MinPrice = int32(price)
		}
	}

	if maxPriceStr != "" {
		if price, err := strconv.Atoi(maxPriceStr); err == nil && price >= 0 {
			req.MaxPrice = int32(price)
		}
	}

	res, err := h.client.FindByMerchant(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationProduct(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List products by category
// List products by category
// List products by category
// @Summary List products by category
// @Tags Product
// @Accept json
// @Produce json
// @Param category_name path string true "category name"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsesProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/category/:category_name [get]
// @Security BearerAuth
func (h *productHandleApi) FindByCategory(w http.ResponseWriter, r *http.Request) error {
	categoryName := chi.URLParam(r, "category_name")
	if categoryName == "" {
		return errors.NewBadRequestError("category name is required")
	}

	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("product:findbycategory:cat_%s:page_%d:size_%d:search_%s", categoryName, page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationProduct](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindAllProductCategoryRequest{
		CategoryName: categoryName,
		Page:         int32(page),
		PageSize:     int32(pageSize),
		Search:       search,
	}

	res, err := h.client.FindByCategory(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationProduct(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get product by ID
// Get product by ID
// Get product by ID
// @Summary Get product by ID
// @Tags Product
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/:id [get]
// @Security BearerAuth
func (h *productHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid product ID")
	}

	cacheKey := fmt.Sprintf("product:findbyid:id_%d", id)
	if cached, found := gateway_cache.Get[response.ApiResponseProduct](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindByIdProductRequest{
		Id: int32(id),
	}

	res, err := h.client.FindById(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProduct(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List active product
// List active product
// List active product
// @Summary List active product
// @Tags Product
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationProductDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/active [get]
// @Security BearerAuth
func (h *productHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("product:findbyactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationProductDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindAllProductRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationProductDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List trashed product
// List trashed product
// List trashed product
// @Summary List trashed product
// @Tags Product
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationProductDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/trashed [get]
// @Security BearerAuth
func (h *productHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("product:findbytrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationProductDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbproduct.FindAllProductRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationProductDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create product
// Create product
// Create product
// @Summary Create product
// @Tags Product
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/create [post]
// @Security BearerAuth
func (h *productHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	formData, err := h.parseProductForm(w, r, true)
	if err != nil {
		return err
	}

	req := &pbproduct.CreateProductRequest{
		MerchantId:   int32(formData.MerchantID),
		CategoryId:   int32(formData.CategoryID),
		Name:         formData.Name,
		Description:  formData.Description,
		Price:        int32(formData.Price),
		CountInStock: int32(formData.CountInStock),
		Brand:        formData.Brand,
		Weight:       int32(formData.Weight),
		ImageProduct: formData.ImagePath,
	}

	res, err := h.client.Create(ctx, req)
	if err != nil {
		if formData.ImagePath != "" {
			h.upload_image.CleanupImageOnFailure(formData.ImagePath)
		}
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProduct(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusCreated, so)
}

// Update product
// Update product
// Update product
// @Summary Update product
// @Tags Product
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseProduct
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/update/:id [post]
// @Security BearerAuth
func (h *productHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	productID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid product ID")
	}

	formData, err := h.parseProductForm(w, r, false)
	if err != nil {
		return err
	}

	req := &pbproduct.UpdateProductRequest{
		ProductId:    int32(productID),
		MerchantId:   int32(formData.MerchantID),
		CategoryId:   int32(formData.CategoryID),
		Name:         formData.Name,
		Description:  formData.Description,
		Price:        int32(formData.Price),
		CountInStock: int32(formData.CountInStock),
		Brand:        formData.Brand,
		Weight:       int32(formData.Weight),
		ImageProduct: formData.ImagePath,
	}

	res, err := h.client.Update(ctx, req)
	if err != nil {
		if formData.ImagePath != "" {
			h.upload_image.CleanupImageOnFailure(formData.ImagePath)
		}
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProduct(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash product
// Trash product
// Trash product
// @Summary Trash product
// @Tags Product
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseProductDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/trashed/:id [post]
// @Security BearerAuth
func (h *productHandleApi) TrashedProduct(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid product ID")
	}

	req := &pbproduct.FindByIdProductRequest{
		Id: int32(id),
	}

	res, err := h.client.TrashedProduct(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsesProductDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore product
// Restore product
// Restore product
// @Summary Restore product
// @Tags Product
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseProductDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/restore/:id [post]
// @Security BearerAuth
func (h *productHandleApi) RestoreProduct(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid product ID")
	}

	req := &pbproduct.FindByIdProductRequest{
		Id: int32(id),
	}

	res, err := h.client.RestoreProduct(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsesProductDeleteAt(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete product permanently
// Delete product permanently
// Delete product permanently
// @Summary Delete product permanently
// @Tags Product
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseProductDelete
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/permanent/:id [delete]
// @Security BearerAuth
func (h *productHandleApi) DeleteProductPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return errors.NewBadRequestError("invalid product ID")
	}

	req := &pbproduct.FindByIdProductRequest{
		Id: int32(id),
	}

	res, err := h.client.DeleteProductPermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProductDelete(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed product
// Restore all trashed product
// Restore all trashed product
// @Summary Restore all trashed product
// @Tags Product
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseProductAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/restore/all [post]
// @Security BearerAuth
func (h *productHandleApi) RestoreAllProduct(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.RestoreAllProduct(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProductAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed product permanently
// Delete all trashed product permanently
// Delete all trashed product permanently
// @Summary Delete all trashed product permanently
// @Tags Product
// @Accept json
// @Produce json
// @Param request body requests.ProductFormData true "Request body"
// @Success 200 {object} response.ApiResponseProductAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/product/permanent/all [post]
// @Security BearerAuth
func (h *productHandleApi) DeleteAllProductPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.DeleteAllProductPermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseProductAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "product:*")
	return httpx.JSON(w, http.StatusOK, so)
}

func (h *productHandleApi) parseProductForm(w http.ResponseWriter, r *http.Request, requireImage bool) (requests.ProductFormData, error) {
	var formData requests.ProductFormData
	var err error

	formData.MerchantID, err = strconv.Atoi(r.FormValue("merchant_id"))
	if err != nil || formData.MerchantID <= 0 {
		return formData, errors.NewBadRequestError("Please provide a valid merchant ID")
	}

	formData.CategoryID, err = strconv.Atoi(r.FormValue("category_id"))
	if err != nil || formData.CategoryID <= 0 {
		return formData, errors.NewBadRequestError("Please provide a valid category ID")
	}

	formData.Name = strings.TrimSpace(r.FormValue("name"))
	if formData.Name == "" {
		return formData, errors.NewBadRequestError("Product name is required")
	}

	formData.Description = strings.TrimSpace(r.FormValue("description"))
	formData.Brand = strings.TrimSpace(r.FormValue("brand"))

	formData.Price, err = strconv.Atoi(r.FormValue("price"))
	if err != nil || formData.Price <= 0 {
		return formData, errors.NewBadRequestError("Please provide a valid positive price")
	}

	formData.CountInStock, err = strconv.Atoi(r.FormValue("count_in_stock"))
	if err != nil || formData.CountInStock < 0 {
		return formData, errors.NewBadRequestError("Please provide a valid stock count (zero or positive)")
	}

	formData.Weight, err = strconv.Atoi(r.FormValue("weight"))
	if err != nil || formData.Weight <= 0 {
		return formData, errors.NewBadRequestError("Please provide a valid positive weight")
	}

	_, file, err := r.FormFile("image_product")
	if err != nil {
		if requireImage {
			return formData, errors.NewBadRequestError("A product image is required")
		}
		return formData, nil
	}

	imagePath, err := h.upload_image.ProcessImageUpload(w, file)
	if err != nil {
		return formData, err
	}

	formData.ImagePath = imagePath
	return formData, nil
}
