package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbrole "github.com/MamangRust/microservice-point-of-sale-pb/role"
	pbuserrole "github.com/MamangRust/microservice-point-of-sale-pb/user_role"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/response"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/emptypb"
)

type roleHandleApi struct {
	client         pbrole.RoleServiceClient
	userRoleClient pbuserrole.UserRoleServiceClient
	logger         logger.LoggerInterface
	mapping        response_api.RoleResponseMapper
	apiHandler     apierror.ApiHandler
	cache          *gateway_cache.GatewayCache
}

func NewHandlerRole(
	router chi.Router,
	client pbrole.RoleServiceClient,
	userRoleClient pbuserrole.UserRoleServiceClient,
	logger logger.LoggerInterface,
	mapping response_api.RoleResponseMapper,
	apiHandler apierror.ApiHandler,
	cache *gateway_cache.GatewayCache,
) *roleHandleApi {
	roleHandler := &roleHandleApi{
		client:         client,
		userRoleClient: userRoleClient,
		logger:         logger,
		mapping:        mapping,
		apiHandler:     apiHandler,
		cache:          cache,
	}

	router.Route("/api/role", func(routerRole chi.Router) {

		routerRole.Get("/", apiHandler.Handle("get-role-findall", roleHandler.FindAll))
		routerRole.Get("/{id}", apiHandler.Handle("get-role-findbyid", roleHandler.FindById))
		routerRole.Get("/active", apiHandler.Handle("get-role-findbyactive", roleHandler.FindByActive))
		routerRole.Get("/trashed", apiHandler.Handle("get-role-findbytrashed", roleHandler.FindByTrashed))
		routerRole.Get("/user/{user_id}", apiHandler.Handle("get-role-findbyuserid", roleHandler.FindByUserId))
		routerRole.Post("/", apiHandler.Handle("post-role-create", roleHandler.Create))
		routerRole.Post("/update/{id}", apiHandler.Handle("post-role-update", roleHandler.Update))
		routerRole.Post("/trashed/{id}", apiHandler.Handle("post-role-trashed", roleHandler.Trashed))
		routerRole.Post("/restore/{id}", apiHandler.Handle("post-role-restore", roleHandler.Restore))
		routerRole.Delete("/permanent/{id}", apiHandler.Handle("delete-role-deletepermanent", roleHandler.DeletePermanent))
		routerRole.Post("/restore/all", apiHandler.Handle("post-role-restoreall", roleHandler.RestoreAll))
		routerRole.Delete("/permanent-all", apiHandler.Handle("delete-role-deleteallpermanent", roleHandler.DeleteAllPermanent))

	})
	return roleHandler
}

// List all role (paginated)
// List all role (paginated)
// List all role (paginated)
// @Summary List all role (paginated)
// @Tags Role
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role [get]
// @Security BearerAuth
func (h *roleHandleApi) FindAll(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("role:findall:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationRole](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbrole.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindAllRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationRole(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get role by ID
// Get role by ID
// Get role by ID
// @Summary Get role by ID
// @Tags Role
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/:id [get]
// @Security BearerAuth
func (h *roleHandleApi) FindById(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("invalid role ID")
	}

	cacheKey := fmt.Sprintf("role:findbyid:id_%d", roleID)
	if cached, found := gateway_cache.Get[response.ApiResponseRole](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbrole.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.client.FindByIdRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRole(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List active role
// List active role
// List active role
// @Summary List active role
// @Tags Role
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRoleDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/active [get]
// @Security BearerAuth
func (h *roleHandleApi) FindByActive(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("role:findbyactive:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationRoleDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbrole.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByActive(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationRoleDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// List trashed role
// List trashed role
// List trashed role
// @Summary List trashed role
// @Tags Role
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Param search query string false "Search keyword"
// @Success 200 {object} response.ApiResponsePaginationRoleDeleteAt
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/trashed [get]
// @Security BearerAuth
func (h *roleHandleApi) FindByTrashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	page := parseQueryInt(r, "page", 1)
	pageSize := parseQueryInt(r, "page_size", 10)
	search := r.URL.Query().Get("search")

	cacheKey := fmt.Sprintf("role:findbytrashed:page_%d:size_%d:search_%s", page, pageSize, search)
	if cached, found := gateway_cache.Get[response.ApiResponsePaginationRoleDeleteAt](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbrole.FindAllRoleRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Search:   search,
	}

	res, err := h.client.FindByTrashed(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsePaginationRoleDeleteAt(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Get role by user ID
// Get role by user ID
// Get role by user ID
// @Summary Get role by user ID
// @Tags Role
// @Accept json
// @Produce json
// @Param user_id path int true "user id"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/user/:user_id [get]
// @Security BearerAuth
func (h *roleHandleApi) FindByUserId(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, err := strconv.Atoi(chi.URLParam(r, "user_id"))
	if err != nil || userID <= 0 {
		return errors.NewBadRequestError("invalid user ID")
	}

	cacheKey := fmt.Sprintf("role:findbyuserid:user_id_%d", userID)
	if cached, found := gateway_cache.Get[response.ApiResponsesRole](ctx, h.cache, cacheKey); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	req := &pbuserrole.FindByIdUserRoleRequest{
		UserId: int32(userID),
	}

	res, err := h.userRoleClient.FindByUserId(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponsesRole(res)

	gateway_cache.Set(ctx, h.cache, cacheKey, so, 5*time.Minute)
	return httpx.JSON(w, http.StatusOK, so)
}

// Create role
// Create role
// Create role
// @Summary Create role
// @Tags Role
// @Accept json
// @Produce json
// @Param request body requests.CreateRoleRequest true "Request body"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role [post]
// @Security BearerAuth
func (h *roleHandleApi) Create(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req requests.CreateRoleRequest
	if err := httpx.Bind(r, &req); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := req.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	reqPb := &pbrole.CreateRoleRequest{
		Name: req.Name,
	}

	res, err := h.client.CreateRole(ctx, reqPb)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRole(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Update role
// Update role
// Update role
// @Summary Update role
// @Tags Role
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Param request body requests.UpdateRoleRequest true "Request body"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/update/:id [post]
// @Security BearerAuth
func (h *roleHandleApi) Update(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("invalid role ID")
	}

	var req requests.UpdateRoleRequest
	if err := httpx.Bind(r, &req); err != nil {
		return errors.NewBadRequestError("bind failed")
	}
	if err := req.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	reqPb := &pbrole.UpdateRoleRequest{
		Id:   int32(roleID),
		Name: req.Name,
	}

	res, err := h.client.UpdateRole(ctx, reqPb)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRole(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Trash role
// Trash role
// Trash role
// @Summary Trash role
// @Tags Role
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/trashed/:id [post]
// @Security BearerAuth
func (h *roleHandleApi) Trashed(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("invalid role ID")
	}

	req := &pbrole.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.client.TrashedRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRole(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore role
// Restore role
// Restore role
// @Summary Restore role
// @Tags Role
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseRole
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/restore/:id [post]
// @Security BearerAuth
func (h *roleHandleApi) Restore(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("invalid role ID")
	}

	req := &pbrole.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.client.RestoreRole(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRole(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete role permanently
// Delete role permanently
// Delete role permanently
// @Summary Delete role permanently
// @Tags Role
// @Accept json
// @Produce json
// @Param id path int true "id"
// @Success 200 {object} response.ApiResponseRoleDelete
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/permanent/:id [delete]
// @Security BearerAuth
func (h *roleHandleApi) DeletePermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	roleID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || roleID <= 0 {
		return errors.NewBadRequestError("invalid role ID")
	}

	req := &pbrole.FindByIdRoleRequest{
		RoleId: int32(roleID),
	}

	res, err := h.client.DeleteRolePermanent(ctx, req)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRoleDelete(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Restore all trashed role
// Restore all trashed role
// Restore all trashed role
// @Summary Restore all trashed role
// @Tags Role
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseRoleAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/restore/all [post]
// @Security BearerAuth
func (h *roleHandleApi) RestoreAll(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.RestoreAllRole(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRoleAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}

// Delete all trashed role permanently
// Delete all trashed role permanently
// Delete all trashed role permanently
// @Summary Delete all trashed role permanently
// @Tags Role
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseRoleAll
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/role/permanent-all [delete]
// @Security BearerAuth
func (h *roleHandleApi) DeleteAllPermanent(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	res, err := h.client.DeleteAllRolePermanent(ctx, &emptypb.Empty{})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToApiResponseRoleAll(res)

	gateway_cache.InvalidatePattern(ctx, h.cache, "role:*")
	return httpx.JSON(w, http.StatusOK, so)
}
