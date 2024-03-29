package repository

import (
	"intern-bcc/domain"

	"gorm.io/gorm"
)

type ICategoryRepository interface {
	GetCategory(category *domain.Categories, categoryParam domain.Categories) error
	CreateCategory(category *domain.Categories) error
}

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) ICategoryRepository {
	return &CategoryRepository{db}
}

func (r *CategoryRepository) GetCategory(category *domain.Categories, categoryParam domain.Categories) error {
	err := r.db.First(category, categoryParam).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *CategoryRepository) CreateCategory(category *domain.Categories) error {
	err := r.db.Create(category).Error
	if err != nil {
		return err
	}

	return nil
}
