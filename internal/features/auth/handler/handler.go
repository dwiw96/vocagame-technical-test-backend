package delivery

import (
	"context"
	"net/http"

	auth "github.com/dwiw96/vocagame-technical-test-backend/internal/features/auth"
	mid "github.com/dwiw96/vocagame-technical-test-backend/pkg/middleware"
	responses "github.com/dwiw96/vocagame-technical-test-backend/pkg/utils/responses"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type authHandler struct {
	router   *gin.Engine
	service  auth.IService
	validate *validator.Validate
	trans    ut.Translator
}

func NewAuthHandler(router *gin.Engine, service auth.IService, pool *pgxpool.Pool, client *redis.Client, ctx context.Context) {
	handler := &authHandler{
		router:   router,
		service:  service,
		validate: validator.New(),
	}

	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(handler.validate, trans)
	handler.trans = trans

	router.Use(mid.AuthMiddleware(ctx, pool, client))

	router.POST("/api/v1/auth/signup", handler.signUp)
	router.POST("/api/v1/auth/login", handler.logIn)
	router.POST("/api/v1/auth/logout", handler.logOut)
	router.DELETE("/api/v1/auth/delete_user", handler.deleteUser)
	router.POST("/api/v1/auth/refresh_token", handler.refreshToken)
}

func translateError(trans ut.Translator, err error) (errTrans []string) {
	errs := err.(validator.ValidationErrors)
	a := (errs.Translate(trans))
	for _, val := range a {
		errTrans = append(errTrans, val)
	}

	return
}

func (d *authHandler) signUp(c *gin.Context) {
	var request signupRequest

	err := c.BindJSON(&request)

	if err != nil {
		c.JSON(422, err.Error())
		responses.ErrorJSON(c, 422, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	err = d.validate.Struct(request)
	if err != nil {
		errTranslated := translateError(d.trans, err)
		responses.ErrorJSON(c, 422, errTranslated, c.Request.RemoteAddr)
		return
	}

	signupInput := toSignUpRequest(request)
	user, _, code, err := d.service.SignUp(signupInput)
	if err != nil {
		responses.ErrorJSON(c, code, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	respBody := toSignUpResponse(user)

	response := responses.SuccessWithDataResponse(respBody, 201, "Sign up success")
	c.IndentedJSON(http.StatusCreated, response)
}

func (d *authHandler) logIn(c *gin.Context) {
	var request signinRequest

	err := c.BindJSON(&request)
	if err != nil {
		c.JSON(422, err)
		return
	}

	err = d.validate.Struct(request)
	if err != nil {
		errTranslated := translateError(d.trans, err)
		responses.ErrorJSON(c, 422, errTranslated, c.Request.RemoteAddr)
		return
	}

	user, accessToken, refreshToken, code, err := d.service.LogIn(auth.LoginRequest(request))
	if err != nil {
		responses.ErrorJSON(c, code, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	respBody := toLoginResponse(user, accessToken, refreshToken)

	response := responses.SuccessWithDataResponse(respBody, 200, "Login success")
	c.IndentedJSON(200, response)
}

func (d *authHandler) logOut(c *gin.Context) {
	authPayload, isExists := c.Keys["payloadKey"].(*auth.JwtPayload)

	if !isExists {
		responses.ErrorJSON(c, 401, []string{"token is wrong"}, c.Request.RemoteAddr)
		return
	}

	err := d.service.LogOut(*authPayload)
	if err != nil {
		responses.ErrorJSON(c, 401, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	c.JSON(200, "logout success")
}

func (d *authHandler) refreshToken(c *gin.Context) {
	var request refreshTokenRequest

	err := c.BindJSON(&request)
	if err != nil {
		return
	}

	err = d.validate.Struct(request)
	if err != nil {
		errTranslated := translateError(d.trans, err)
		responses.ErrorJSON(c, 422, errTranslated, c.Request.RemoteAddr)
		return
	}

	newRefreshToken, newAccessToken, code, err := d.service.RefreshToken(request.RefreshToken, request.AccessToken)
	if err != nil {
		responses.ErrorJSON(c, code, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	respBody := refreshTokenResponse{
		RefreshToken: newRefreshToken,
		AccessToken:  newAccessToken,
	}

	response := responses.SuccessWithDataResponse(respBody, 200, "refresh token success")
	c.IndentedJSON(200, response)
}

func (d *authHandler) deleteUser(c *gin.Context) {
	authPayload, isExists := c.Keys["payloadKey"].(*auth.JwtPayload)

	if !isExists {
		responses.ErrorJSON(c, 401, []string{"token is wrong"}, c.Request.RemoteAddr)
		return
	}

	arg := auth.DeleteUserParams{
		ID:    authPayload.UserID,
		Email: authPayload.Email,
	}
	code, err := d.service.DeleteUser(arg)
	if err != nil {
		responses.ErrorJSON(c, code, []string{err.Error()}, c.Request.RemoteAddr)
		return
	}

	response := responses.SuccessResponse("user deleted")
	c.IndentedJSON(code, response)
}