package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	auth_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/auth"
	"github.com/MamangRust/microservice-point-of-sale-apigateway/httpx"
	pbauth "github.com/MamangRust/microservice-point-of-sale-pb/auth"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-shared/domain/requests"
	"github.com/MamangRust/microservice-point-of-sale-shared/errors"
	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
	"github.com/go-chi/chi/v5"
)

type authHandleApi struct {
	client     pbauth.AuthServiceClient
	logger     logger.LoggerInterface
	mapping    response_api.AuthResponseMapper
	apiHandler apierror.ApiHandler
	cache      auth_cache.AuthMencache
}

func NewHandlerAuth(
	router chi.Router,
	client pbauth.AuthServiceClient,
	logger logger.LoggerInterface,
	mapper response_api.AuthResponseMapper,
	apiHandler apierror.ApiHandler,
	cache auth_cache.AuthMencache,
) *authHandleApi {
	authHandler := &authHandleApi{
		client:     client,
		logger:     logger,
		mapping:    mapper,
		apiHandler: apiHandler,
		cache:      cache,
	}

	router.Route("/api/auth", func(routerAuth chi.Router) {

		routerAuth.Get("/verify-code", apiHandler.Handle("get-auth-verifycode", authHandler.VerifyCode))
		routerAuth.Post("/forgot-password", apiHandler.Handle("post-auth-forgotpassword", authHandler.ForgotPassword))
		routerAuth.Post("/reset-password", apiHandler.Handle("post-auth-resetpassword", authHandler.ResetPassword))
		routerAuth.Get("/hello", apiHandler.Handle("get-auth-handlehello", authHandler.HandleHello))
		routerAuth.Post("/register", apiHandler.Handle("post-auth-register", authHandler.Register))
		routerAuth.Post("/login", apiHandler.Handle("post-auth-login", authHandler.Login))
		routerAuth.Post("/refresh-token", apiHandler.Handle("post-auth-refreshtoken", authHandler.RefreshToken))
		routerAuth.Get("/me", apiHandler.Handle("get-auth-getme", authHandler.GetMe))

	})
	return authHandler
}

// Health check
// Health check
// Health check
// @Summary Health check
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/hello [get]
func (h *authHandleApi) HandleHello(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello"))
	return nil
}

// Verify reset code
// Verify reset code
// Verify reset code
// @Summary Verify reset code
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseVerifyCode
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/verify-code [get]
func (h *authHandleApi) VerifyCode(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	verifyCode, err := parseQueryStringRequired(r, "verify_code")
	if err != nil {
		return errors.NewBadRequestError("invalid verify_code")
	}

	res, err := h.client.VerifyCode(ctx, &pbauth.VerifyCodeRequest{
		Code: verifyCode,
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToResponseVerifyCode(res)
	return httpx.JSON(w, http.StatusOK, so)
}

// Request password reset
// Request password reset
// Request password reset
// @Summary Request password reset
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body requests.ForgotPasswordRequest true "Request body"
// @Success 200 {object} response.ApiResponseForgotPassword
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/forgot-password [post]
func (h *authHandleApi) ForgotPassword(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var body requests.ForgotPasswordRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}

	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	res, err := h.client.ForgotPassword(ctx, &pbauth.ForgotPasswordRequest{
		Email: body.Email,
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	resp := h.mapping.ToResponseForgotPassword(res)
	return httpx.JSON(w, http.StatusOK, resp)
}

// Reset password
// Reset password
// Reset password
// @Summary Reset password
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body requests.CreateResetPasswordRequest true "Request body"
// @Success 200 {object} response.ApiResponseResetPassword
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/reset-password [post]
func (h *authHandleApi) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var body requests.CreateResetPasswordRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}

	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	res, err := h.client.ResetPassword(ctx, &pbauth.ResetPasswordRequest{
		ResetToken:      body.ResetToken,
		Password:        body.Password,
		ConfirmPassword: body.ConfirmPassword,
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToResponseResetPassword(res)
	return httpx.JSON(w, http.StatusOK, so)
}

// Register a new user
// Register a new user
// Register a new user
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body requests.CreateUserRequest true "Request body"
// @Success 200 {object} response.ApiResponseRegister
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/register [post]
func (h *authHandleApi) Register(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var body requests.CreateUserRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}

	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	data := &pbauth.RegisterRequest{
		Firstname:       body.FirstName,
		Lastname:        body.LastName,
		Email:           body.Email,
		Password:        body.Password,
		ConfirmPassword: body.ConfirmPassword,
	}

	res, err := h.client.RegisterUser(ctx, data)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToResponseRegister(res)
	return httpx.JSON(w, http.StatusOK, so)
}

// Login
// Login
// Login
// @Summary Login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body requests.AuthRequest true "Request body"
// @Success 200 {object} response.ApiResponseLogin
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/login [post]
func (h *authHandleApi) Login(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var body requests.AuthRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}

	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	// Login Cache Check
	if cached, found := h.cache.GetCachedLogin(ctx, body.Email); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	data := &pbauth.LoginRequest{
		Email:    body.Email,
		Password: body.Password,
	}

	res, err := h.client.LoginUser(ctx, data)
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	mappedResponse := h.mapping.ToResponseLogin(res)

	// Set Login Cache
	h.cache.SetCachedLogin(ctx, body.Email, mappedResponse)

	return httpx.JSON(w, http.StatusOK, mappedResponse)
}

// Refresh access token
// Refresh access token
// Refresh access token
// @Summary Refresh access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body requests.RefreshTokenRequest true "Request body"
// @Success 200 {object} response.ApiResponseRefreshToken
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/refresh-token [post]
func (h *authHandleApi) RefreshToken(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var body requests.RefreshTokenRequest
	if err := httpx.Bind(r, &body); err != nil {
		return errors.NewBadRequestError("bind failed")
	}

	if err := body.Validate(); err != nil {
		return errors.NewBadRequestError("validation failed")
	}

	// Refresh Token Cache Check
	if cached, found := h.cache.GetRefreshToken(ctx, body.RefreshToken); found && cached != nil {
		return httpx.JSON(w, http.StatusOK, cached)
	}

	res, err := h.client.RefreshToken(ctx, &pbauth.RefreshTokenRequest{
		RefreshToken: body.RefreshToken,
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToResponseRefreshToken(res)

	// Set Refresh Token Cache
	h.cache.SetRefreshToken(ctx, body.RefreshToken, so)

	return httpx.JSON(w, http.StatusOK, so)
}

// Get current user profile
// Get current user profile
// Get current user profile
// @Summary Get current user profile
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponseGetMe
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/auth/me [get]
// @Security BearerAuth
func (h *authHandleApi) GetMe(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return errors.ErrUnauthorized.WithMessage("invalid authorization header")
	}

	// Extract userID from context if set by auth middleware
	var userID string
	if uVal := httpx.Get(r, "userID"); uVal != nil {
		if uStr, ok := uVal.(string); ok {
			userID = uStr
		}
	}

	// User Info Cache Check
	if userID != "" {
		if cached, found := h.cache.GetCachedUserInfo(ctx, userID); found && cached != nil {
			return httpx.JSON(w, http.StatusOK, cached)
		}
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")
	res, err := h.client.GetMe(ctx, &pbauth.GetMeRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		return errors.ParseGrpcError(err)
	}

	so := h.mapping.ToResponseGetMe(res)

	// Set User Info Cache
	if userID == "" && so.Data != nil {
		userID = strconv.Itoa(so.Data.ID)
	}
	if userID != "" {
		h.cache.SetCachedUserInfo(ctx, userID, so)
	}

	return httpx.JSON(w, http.StatusOK, so)
}

func parseQueryStringRequired(r *http.Request, name string) (string, error) {
	val := r.URL.Query().Get(name)
	if val == "" {
		return "", errors.NewBadRequestError("missing " + name)
	}
	return val, nil
}
