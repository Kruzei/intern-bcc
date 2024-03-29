package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"intern-bcc/domain"
	"intern-bcc/pkg/redis"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IProductRepository interface {
	GetProduct(product *domain.Products, productParam domain.ProductParam) error
	GetProducts(c *gin.Context, ctx context.Context, product *[]domain.Products, productParam domain.ProductParam) error
	GetTotalProduct(totalProduct *int64) error
	CreateProduct(newProduct *domain.Products) error
	UpdateProduct(product *domain.ProductUpdate, productId uuid.UUID) error
}

type ProductRepository struct {
	db    *gorm.DB
	redis redis.IRedis
}

func NewProductRepository(db *gorm.DB, redis redis.IRedis) IProductRepository {
	return &ProductRepository{db, redis}
}

func (r *ProductRepository) GetProducts(c *gin.Context, ctx context.Context, product *[]domain.Products, productParam domain.ProductParam) error {
	byteParam, err := json.Marshal(productParam)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(KeySetProducts, string(byteParam))
	stringData, err := r.redis.GetRedis(ctx, key)
	if err != nil {
		err := r.db.
			Joins("JOIN merchants ON merchants.id = products.merchant_id").
			Joins("JOIN universities ON universities.id = merchants.university_id").
			Joins("JOIN provinces ON provinces.id = merchants.province_id").
			Where("IF(? != 0, universities.id = ?, 1) AND IF(? != 0, provinces.id = ?, 1)", productParam.UniversityId, productParam.UniversityId, productParam.ProvinceId, productParam.ProvinceId).
			Limit(Limit).
			Offset(productParam.Offset).
			Preload("Merchant.University").
			Preload("Merchant.Province").
			Find(&product, productParam).Error
		if err != nil {
			return err
		}

		byteProduct, err := json.Marshal(product)
		if err != nil {
			return err
		}

		err = r.redis.SetRedis(ctx, key, string(byteProduct), 5*time.Minute)
		if err != nil {
			return err
		}

		return nil
	}

	err = json.Unmarshal([]byte(stringData), &product)
	if err != nil {
		return err
	}
	return nil
}

func (r *ProductRepository) GetProduct(product *domain.Products, productParam domain.ProductParam) error {
	err := r.db.Preload("Category").Preload("Merchant.University").Preload("Merchant.Province").First(product, productParam).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ProductRepository) GetTotalProduct(totalProduct *int64) error {
	err := r.db.Model(domain.Products{}).Count(totalProduct).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ProductRepository) CreateProduct(newProduct *domain.Products) error {
	err := r.db.Create(newProduct).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *ProductRepository) UpdateProduct(product *domain.ProductUpdate, productId uuid.UUID) error {
	err := r.db.Model(domain.Products{}).Where("id = ?", productId).Updates(product).Error
	if err != nil {
		return err
	}

	return nil
}
