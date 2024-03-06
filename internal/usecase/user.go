package usecase

import (
	"intern-bcc/domain"
	"intern-bcc/internal/repository"
	"intern-bcc/pkg/jwt"
	"intern-bcc/pkg/response"
	"net/http"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type IUserUsecase interface {
	Register(userRequest domain.UserRequest) any
	Login(userLogin domain.UserLogin) (domain.LoginResponse, any)
}

type UserUsecase struct {
	userRepository repository.IUserRepository
	jwt            jwt.IJwt
}

func NewUserUsecase(userRepository repository.IUserRepository, jwt jwt.IJwt) IUserUsecase {
	return &UserUsecase{
		userRepository: userRepository,
		jwt:            jwt,
	}
}

func (u *UserUsecase) Register(userRequest domain.UserRequest) any {
	hashPassword, err := bcrypt.GenerateFromPassword([]byte(userRequest.Password), 10)
	if err != nil {
		return response.ErrorObject{
			Code:    http.StatusInternalServerError,
			Err:     err,
			Message: "error when hashing password",
		}
	}

	NewUser := domain.Users{
		Id:       uuid.New(),
		Username: userRequest.Username,
		Email:    userRequest.Email,
		Password: string(hashPassword),
	}

	err = u.userRepository.Register(&NewUser)
	if err != nil {
		return response.ErrorObject{
			Code:    http.StatusInternalServerError,
			Err:     err,
			Message: "error occured when creating user",
		}
	}

	return nil
}

func (u *UserUsecase) Login(userLogin domain.UserLogin) (domain.LoginResponse, any) {
	var user domain.Users
	err := u.userRepository.GetUser(&user, domain.UserParam{
		Email: userLogin.Email,
	})
	if err != nil {
		return domain.LoginResponse{}, response.ErrorObject{
			Code:    http.StatusNotFound,
			Message: "email or password invalid",
			Err:     err,
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userLogin.Password))
	if err != nil {
		return domain.LoginResponse{}, response.ErrorObject{
			Code:    http.StatusNotFound,
			Message: "email or password invalid",
			Err:     err,
		}
	}

	tokenString, err := u.jwt.GenerateToken(user.Id)
	if err != nil {
		return domain.LoginResponse{}, response.ErrorObject{
			Code:    http.StatusInternalServerError,
			Message: "faield to generate jwt token",
			Err:     err,
		}
	}

	loginUser := domain.LoginResponse{
		JWT: tokenString,
	}

	return loginUser, nil
}
