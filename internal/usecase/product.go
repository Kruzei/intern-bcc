package usecase

import (
	"context"
	"errors"
	"fmt"
	"intern-bcc/domain"
	"intern-bcc/internal/repository"
	"intern-bcc/pkg/jwt"
	"intern-bcc/pkg/response"
	"intern-bcc/pkg/supabase"
	"math"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IProductUsecase interface {
	GetProduct(productParam domain.ProductParam) (domain.ProductResponse, error)
	GetProducts(c *gin.Context, ctx context.Context, productParam domain.ProductParam) ([]domain.ProductResponses, error)
	GetOwnProduct(productParam domain.ProductParam) (domain.ProductProfileResponse, error)
	CreateProduct(c *gin.Context, productRequest domain.ProductRequest) error
	UpdateProduct(c *gin.Context, productId uuid.UUID, updateProduct domain.ProductUpdate) (domain.ProductProfileResponse, error)
	UploadProductPhoto(c *gin.Context, productId uuid.UUID, productPhoto *multipart.FileHeader) (domain.ProductProfileResponse, error)
}

type ProductUsecase struct {
	productRepository  repository.IProductRepository
	merchantRepository repository.IMerchantRepository
	categoryRepository repository.ICategoryRepository
	jwt                jwt.IJwt
	supabase           supabase.ISupabase
}

func NewProductUsecase(productRepository repository.IProductRepository, jwt jwt.IJwt,
	merchantRepository repository.IMerchantRepository, categoryRepository repository.ICategoryRepository,
	supabase supabase.ISupabase) IProductUsecase {
	return &ProductUsecase{
		productRepository:  productRepository,
		jwt:                jwt,
		merchantRepository: merchantRepository,
		categoryRepository: categoryRepository,
		supabase:           supabase,
	}
}

func (u *ProductUsecase) GetProduct(productParam domain.ProductParam) (domain.ProductResponse, error) {
	var product domain.Products
	err := u.productRepository.GetProduct(&product, productParam)
	if err != nil {
		return domain.ProductResponse{}, response.NewError(http.StatusNotFound, "an error occured when get product", err)
	}

	countryPhoneNumber := strings.Replace(product.Merchant.PhoneNumber, "0", "+62", 1)
	linkWhatsApp := fmt.Sprintf("https://wa.me/%v", countryPhoneNumber)

	productResponse := domain.ProductResponse{
		Id:           product.Id,
		Name:         product.Name,
		Description:  product.Description,
		Price:        product.Price,
		ProductPhoto: product.ProductPhoto,
		MerchantName: product.Merchant.MerchantName,
		University:   product.Merchant.University.University,
		Faculty:      product.Merchant.Faculty,
		Province:     product.Merchant.Province.Province,
		City:         product.Merchant.City,
		WhatsApp:     linkWhatsApp,
		Instagram:    product.Merchant.Instagram,
	}

	return productResponse, nil
}

func (u *ProductUsecase) GetProducts(c *gin.Context, ctx context.Context, productParam domain.ProductParam) ([]domain.ProductResponses, error) {
	if productParam.Page <= 0 {
		productParam.Page = 1
	}

	offSet := (productParam.Page - 1) * 6
	productParam.Offset = offSet

	var totalProduct int64
	err := u.productRepository.GetTotalProduct(&totalProduct)
	if err != nil {
		return []domain.ProductResponses{}, response.NewError(http.StatusInternalServerError, "failed to get total product", err)
	}

	totalPage := (int)(math.Ceil(float64(totalProduct) / 6))
	if productParam.Page > totalPage {
		return []domain.ProductResponses{}, response.NewError(http.StatusBadRequest, "can not find page", errors.New("request page bigger than maximum page"))
	}

	var products []domain.Products

	err = u.productRepository.GetProducts(c, ctx, &products, productParam)
	if err != nil {
		return []domain.ProductResponses{}, response.NewError(http.StatusInternalServerError, "failed to get products", err)
	}

	var productResponses []domain.ProductResponses
	for _, p := range products {
		productResponse := domain.ProductResponses{
			Id:           p.Id,
			Name:         p.Name,
			MerchantName: p.Merchant.MerchantName,
			University:   p.Merchant.University.University,
			Price:        p.Price,
			ProductPhoto: p.ProductPhoto,
		}

		productResponses = append(productResponses, productResponse)
	}

	return productResponses, nil
}

func (u *ProductUsecase) GetOwnProduct(productParam domain.ProductParam) (domain.ProductProfileResponse, error) {
	var product domain.Products
	err := u.productRepository.GetProduct(&product, productParam)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "an error occured when get product", err)
	}

	productResponse := domain.ProductProfileResponse{
		Id:           product.Id,
		Name:         product.Name,
		Description:  product.Description,
		Category:     product.Category.Category,
		Price:        product.Price,
		ProductPhoto: product.ProductPhoto,
	}

	return productResponse, nil
}

func (u *ProductUsecase) CreateProduct(c *gin.Context, productRequest domain.ProductRequest) error {
	user, err := u.jwt.GetLoginUser(c)
	if err != nil {
		return response.NewError(http.StatusNotFound, "an error occured when get login user", err)
	}

	var merchant domain.Merchants
	err = u.merchantRepository.GetMerchant(&merchant, domain.MerchantParam{UserId: user.Id})
	if err != nil {
		return response.NewError(http.StatusNotFound, "an error occured when get merchant", err)
	}

	if !merchant.IsActive {
		return response.NewError(http.StatusNotFound, "failed to create product", errors.New("please verify your merchant"))
	}

	var category domain.Categories
	err = u.categoryRepository.GetCategory(&category, domain.Categories{Id: productRequest.Category})
	if err != nil {
		return response.NewError(http.StatusNotFound, "category not found", err)
	}

	if category.Id > 6 {
		return response.NewError(http.StatusBadRequest, "can no use this category for product", errors.New("can not use information category"))

	}

	newProduct := domain.Products{
		Id:          uuid.New(),
		Name:        productRequest.Name,
		MerchantId:  merchant.Id,
		Description: productRequest.Description,
		Price:       productRequest.Price,
		CategoryId:  category.Id,
	}

	err = u.productRepository.CreateProduct(&newProduct)
	if err != nil {
		return response.NewError(http.StatusInternalServerError, "an error occured when creating product", err)
	}

	return nil
}

func (u *ProductUsecase) UpdateProduct(c *gin.Context, productId uuid.UUID, updateProduct domain.ProductUpdate) (domain.ProductProfileResponse, error) {
	user, err := u.jwt.GetLoginUser(c)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "an error occured when get login user", err)
	}

	var product domain.Products
	err = u.productRepository.GetProduct(&product, domain.ProductParam{Id: productId})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "an error occured when get product", err)
	}

	var category domain.Categories
	err = u.categoryRepository.GetCategory(&category, domain.Categories{Id: updateProduct.Category})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "category not found", err)
	}

	if category.Id > 6 {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusBadRequest, "can no use this category for product", errors.New("can not use information category"))
	}

	var merchant domain.Merchants
	err = u.merchantRepository.GetMerchant(&merchant, domain.MerchantParam{UserId: user.Id})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "merchant not found", err)
	}

	if merchant.Id != product.MerchantId {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusUnauthorized, "access denied", errors.New("can not edit other people merchant"))
	}

	err = u.productRepository.UpdateProduct(&updateProduct, product.Id)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "an error occured when update product", err)
	}

	var updatedProduct domain.Products
	err = u.productRepository.GetProduct(&updatedProduct, domain.ProductParam{Id: product.Id})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "an error occured when get updated product", err)
	}

	updatedProductResponse := domain.ProductProfileResponse{
		Id:           updatedProduct.Id,
		Name:         updatedProduct.Name,
		Description:  updatedProduct.Description,
		Price:        updatedProduct.Price,
		ProductPhoto: updatedProduct.ProductPhoto,
		Category:     updatedProduct.Category.Category,
	}

	return updatedProductResponse, nil
}

func (u *ProductUsecase) UploadProductPhoto(c *gin.Context, productId uuid.UUID, productPhoto *multipart.FileHeader) (domain.ProductProfileResponse, error) {
	user, err := u.jwt.GetLoginUser(c)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "an error occured when get login user", err)
	}

	var product domain.Products
	err = u.productRepository.GetProduct(&product, domain.ProductParam{Id: productId})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "an error occured when get product", err)
	}

	var merchant domain.Merchants
	err = u.merchantRepository.GetMerchant(&merchant, domain.MerchantParam{UserId: user.Id})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusNotFound, "merchant not found", err)
	}

	if merchant.Id != product.MerchantId {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusUnauthorized, "access denied", errors.New("can not edit other people merchant"))
	}

	if product.ProductPhoto != "" {
		err = u.supabase.Delete(product.ProductPhoto)
		if err != nil {
			return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "error occured when deleting old product photo", err)
		}
	}

	productPhoto.Filename = fmt.Sprintf("%v-%v", time.Now().String(), productPhoto.Filename)
	productPhoto.Filename = strings.Replace(productPhoto.Filename, " ", "-", -1)

	newProductPhoto, err := u.supabase.Upload(productPhoto)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "failed to upload photo", err)
	}

	err = u.productRepository.UpdateProduct(&domain.ProductUpdate{ProductPhoto: newProductPhoto}, product.Id)
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "an error occured when update product", err)
	}

	var updatedProduct domain.Products
	err = u.productRepository.GetProduct(&updatedProduct, domain.ProductParam{Id: product.Id})
	if err != nil {
		return domain.ProductProfileResponse{}, response.NewError(http.StatusInternalServerError, "an error occured when get updated product", err)
	}

	updatedProductResponse := domain.ProductProfileResponse{
		Id:           updatedProduct.Id,
		Name:         updatedProduct.Name,
		Description:  updatedProduct.Description,
		Price:        updatedProduct.Price,
		ProductPhoto: updatedProduct.ProductPhoto,
		Category:     updatedProduct.Category.Category,
	}

	return updatedProductResponse, nil
}
