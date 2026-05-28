package handler

import (
	"errors"
	"gymlink/internal/apperrs"
	"gymlink/internal/dto"
	"gymlink/internal/service"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// handler が何に依存するかを明示
type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) SignUpUserById(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	token := strings.TrimPrefix(authz, "Bearer ")

	//　requestBody の読み取り
	var userCreate dto.UserCreateType
	if err := ctx.ShouldBindJSON(&userCreate); err != nil {
		log.Println("error: user data bind")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "server internal error"})
		return
	}

	// service に依存
	err := h.svc.SignUpUser(ctx.Request.Context(), userCreate.Name, userCreate.AvatarUrl, token)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Finish Setting Endpoints Successfully✅"})
}

func (h *UserHandler) LoginUser(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		log.Println("login error: missing bearer token, Authorization:", authz)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	token := strings.TrimPrefix(authz, "Bearer ")
	log.Println("login attempt: token prefix =", token[:min(len(token), 20)])

	userRaw, err := h.svc.LoginUser(ctx.Request.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("login error: firebase token verification failed:", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			log.Println("login error: internal error:", err)
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	user := dto.UserJsonType{
		Id:   userRaw.Id,
		Name: userRaw.Name,
	}
	ctx.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetProfilebyId(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	idStr := ctx.Param("user_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		log.Println("error: invalid user id")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	profileRaw, err := h.svc.GetProfile(ctx.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	profile := dto.ProfileType{
		Id:            profileRaw.Id,
		Name:          profileRaw.Name,
		ProfileImage:  profileRaw.ProfileImage,
		FollowCount:   profileRaw.FollowCount,
		FollowerCount: profileRaw.FollowerCount,
	}

	ctx.JSON(http.StatusOK, profile)
}

func (h *UserHandler) FollowUser(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}

	//　requestBody の読み取り
	var userFollow dto.UserFollowType
	if err := ctx.ShouldBindJSON(&userFollow); err != nil {
		log.Println("error: user follow")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "server internal error"})
		return
	}
	token := strings.TrimPrefix(authz, "Bearer ")

	err := h.svc.FollowUser(ctx.Request.Context(), userFollow.FollowerId, userFollow.FollowedId, token)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user follow successfully"})
}

func (h *UserHandler) GetFollowing(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing Bearer token"})
		return
	}

	idStr := ctx.Param("user_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		log.Println("error: invalid user")
	}

	followingUsersRaw, err := h.svc.GetFollowingById(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	followingUsers := []dto.UserJsonType{}
	for _, user := range followingUsersRaw {
		followingUsers = append(followingUsers, dto.UserJsonType{
			Id:   user.Id,
			Name: user.Name,
		})
	}
	ctx.JSON(http.StatusOK, followingUsers)
}

func (h *UserHandler) GetFollowed(ctx *gin.Context) {
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing bearer token"})
		return
	}

	idStr := ctx.Param("user_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		log.Println("error: invalid user")
	}

	followedUsersRaw, err := h.svc.GetFollowedById(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	followedUsers := []dto.UserJsonType{}
	for _, user := range followedUsersRaw {
		followedUsers = append(followedUsers, dto.UserJsonType{
			Id:   user.Id,
			Name: user.Name,
		})
	}
	ctx.JSON(http.StatusOK, followedUsers)
}

func (h *UserHandler) CheckFollow(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	token := strings.TrimPrefix(authz, "Bearer ")

	idStr := ctx.Param("user_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		log.Println("error: invalid user id")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	followed, err := h.svc.CheckFollowById(ctx.Request.Context(), id, token)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"followed": followed})

}

func (h *UserHandler) DeleteFollowUser(ctx *gin.Context) {
	// ヘッダー取り出し
	authz := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
		return
	}
	token := strings.TrimPrefix(authz, "Bearer ")
	//　requestBody の読み取り
	var userDeleteFollow dto.UserFollowType
	if err := ctx.ShouldBindJSON(&userDeleteFollow); err != nil {
		log.Println("error: user follow")
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "server internal error"})
		return
	}

	err := h.svc.DeleteFollowUser(ctx.Request.Context(), userDeleteFollow.FollowerId, userDeleteFollow.FollowedId, token)
	if err != nil {
		switch {
		case errors.Is(err, apperrs.ErrVerifyUser):
			log.Println("failed to verify user : ", err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user request"})
			return
		default:
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "user follow successfully"})
}
