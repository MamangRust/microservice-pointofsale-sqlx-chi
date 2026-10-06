package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MamangRust/microservice-point-of-sale-apigateway/apierror"
	auth_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/auth"
	gateway_cache "github.com/MamangRust/microservice-point-of-sale-apigateway/cache/gateway_cache"
	pbauth "github.com/MamangRust/microservice-point-of-sale-pb/auth"
	pbcashier "github.com/MamangRust/microservice-point-of-sale-pb/cashier"
	pbcategory "github.com/MamangRust/microservice-point-of-sale-pb/category"
	pbmerchant "github.com/MamangRust/microservice-point-of-sale-pb/merchant"
	pbmerchantdoc "github.com/MamangRust/microservice-point-of-sale-pb/merchant_document"
	pborder "github.com/MamangRust/microservice-point-of-sale-pb/order"
	pborderitem "github.com/MamangRust/microservice-point-of-sale-pb/order_item"
	pbproduct "github.com/MamangRust/microservice-point-of-sale-pb/product"
	pbrole "github.com/MamangRust/microservice-point-of-sale-pb/role"
	pbtransaction "github.com/MamangRust/microservice-point-of-sale-pb/transaction"
	pbuser "github.com/MamangRust/microservice-point-of-sale-pb/user"
	pbuserrole "github.com/MamangRust/microservice-point-of-sale-pb/user_role"
	"github.com/MamangRust/microservice-point-of-sale-pkg/auth"
	"github.com/MamangRust/microservice-point-of-sale-pkg/logger"
	"github.com/MamangRust/microservice-point-of-sale-pkg/upload_image"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"

	response_api "github.com/MamangRust/microservice-point-of-sale-shared/mapper/response/api"
)

type ServiceConnections struct {
	Auth        *grpc.ClientConn
	Role        *grpc.ClientConn
	User        *grpc.ClientConn
	Cashier     *grpc.ClientConn
	Category    *grpc.ClientConn
	Merchant    *grpc.ClientConn
	OrderItem   *grpc.ClientConn
	Order       *grpc.ClientConn
	Product     *grpc.ClientConn
	Transaction *grpc.ClientConn
	StatsReader *grpc.ClientConn
}

type Deps struct {
	Token              auth.TokenManager
	Router             chi.Router
	Logger             logger.LoggerInterface
	Mapping            *response_api.ResponseApiMapper
	ImageUpload        upload_image.ImageUploads
	ServiceConnections *ServiceConnections
	ApiHandler         apierror.ApiHandler
	AuthCache          auth_cache.AuthMencache
	GatewayCache       *gateway_cache.GatewayCache
}

func NewHandler(deps *Deps) {
	clientAuth := pbauth.NewAuthServiceClient(deps.ServiceConnections.Auth)
	clientRole := pbrole.NewRoleServiceClient(deps.ServiceConnections.Role)
	clientUserRole := pbuserrole.NewUserRoleServiceClient(deps.ServiceConnections.Role)
	clientUser := pbuser.NewUserServiceClient(deps.ServiceConnections.User)
	clientCategory := pbcategory.NewCategoryServiceClient(deps.ServiceConnections.Category)
	clientCashier := pbcashier.NewCashierServiceClient(deps.ServiceConnections.Cashier)
	clientMerchant := pbmerchant.NewMerchantServiceClient(deps.ServiceConnections.Merchant)
	clientMerchantDocument := pbmerchantdoc.NewMerchantDocumentServiceClient(deps.ServiceConnections.Merchant)
	clientOrderItem := pborderitem.NewOrderItemServiceClient(deps.ServiceConnections.OrderItem)
	clientOrder := pborder.NewOrderServiceClient(deps.ServiceConnections.Order)
	clientProduct := pbproduct.NewProductServiceClient(deps.ServiceConnections.Product)
	clientTransaction := pbtransaction.NewTransactionServiceClient(deps.ServiceConnections.Transaction)

	NewHandlerAuth(deps.Router, clientAuth, deps.Logger, deps.Mapping.AuthResponseMapper, deps.ApiHandler, deps.AuthCache)
	NewHandlerRole(deps.Router, clientRole, clientUserRole, deps.Logger, deps.Mapping.RoleResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerUser(deps.Router, clientUser, deps.Logger, deps.Mapping.UserResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerCategory(deps.Router, clientCategory, deps.Logger, deps.Mapping.CategoryResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerCashier(deps.Router, clientCashier, deps.Logger, deps.Mapping.CashierResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerMerchant(deps.Router, clientMerchant, deps.Logger, deps.Mapping.MerchantResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerMerchantDocument(deps.Router, clientMerchantDocument, deps.Logger, deps.Mapping.MerchantDocumentProMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerOrderItem(deps.Router, clientOrderItem, deps.Logger, deps.Mapping.OrderItemResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerOrder(deps.Router, clientOrder, deps.Logger, deps.Mapping.OrderResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerProduct(deps.Router, clientProduct, deps.Logger, deps.Mapping.ProductResponseMapper, deps.ImageUpload, deps.ApiHandler, deps.GatewayCache)
	NewHandlerTransaction(deps.Router, clientTransaction, deps.Logger, deps.Mapping.TransactionResponseMapper, deps.ApiHandler, deps.GatewayCache)
	NewHandlerStats(deps.Router, deps.ServiceConnections.StatsReader, deps.Logger, deps.Mapping.StatsResponseMapper, deps.ApiHandler, deps.GatewayCache)
}

func parseQueryInt(r *http.Request, key string, defaultValue int) int {
	val, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || val <= 0 {
		return defaultValue
	}
	return val
}

func parseQueryIntWithValidation(r *http.Request, key string, min, max int) (int, error) {
	valStr := r.URL.Query().Get(key)
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", key)
	}
	if val < min || val > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return val, nil
}
