package repository

import (
	"intern-bcc/domain"

	"gorm.io/gorm"
)

type IInformationRepository interface{
	GetInformation(information *domain.Information, informationParam domain.Information) error
	CreateInformation(newInformation *domain.Information) error
	UpdateInformation(information *domain.InformationUpdate, informationId int) error
}

type InformationRepository struct {
	db *gorm.DB
}

func NewInformationRepository(db *gorm.DB) IInformationRepository {
	return &InformationRepository{db}
}

func(r *InformationRepository) GetInformation(information *domain.Information, informationParam domain.Information) error {
	err := r.db.First(information, informationParam).Error
	if err != nil {
		return nil
	}

	return err
}

func (r *InformationRepository) CreateInformation(newInformation *domain.Information) error {
	tx := r.db.Begin()

	err := r.db.Create(newInformation).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (r *InformationRepository) UpdateInformation(information *domain.InformationUpdate, informationId int) error {
	err := r.db.Where("id = ?", informationId).Updates(information).Error
	if err != nil {
		return err
	}

	return nil
}
