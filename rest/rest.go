package rest

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	// import to make swagger docs accessible
	_ "github.com/refundable-tgm/huginn/docs"
	ginSwagger "github.com/swaggo/gin-swagger"   // gin swagger middleware
	"github.com/swaggo/gin-swagger/swaggerFiles" // swagger files
	"log"
	"net/http"
	"os"
	"strconv"
)

// Port is the port this api will listen to
const Port = 8080

// DebugFilePath to where a .debug file lies
const DebugFilePath = "/vol/files/.debug"

// StartService starts the rest service
// @title Refundable
// @version 1.1
// @description This REST-API provides the backend of Refundable
// @contact.name Michael Beier - Entwickler
// @contact.url https://mbeier.at
// @contact.email admin@mbeier.at
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @license.name MIT License
// @license.url https://github.com/refundable-tgm/huginn/blob/master/LICENSE
// @host localhost:8080
// @BasePath /api
// @query.collection.format multi
func StartService() {
	// initializing Token Manager
	InitTokenManager()

	// Setting Mode of API
	if debugMode() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Creating new Router
	router := gin.Default()

	// Handling CORS Requests
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AddAllowHeaders("Authorization")
	router.Use(cors.New(config))

	// Registering routes under API Group
	api := router.Group("/api")
	{
		api.POST("/login", Login)
		api.POST("/logout", AuthWall(), Logout)
		api.POST("/login/refresh", Refresh)
		api.GET("/getTeacherByShort", AuthWall(), GetTeacherByShort)
		api.GET("/getTeacher", AuthWall(), GetTeacher)
		api.GET("/getTeacherByUntis", AuthWall(), GetTeacherByUntis)
		api.POST("/setTeacherPermissions", AuthWall(), SetTeacherPermissions)
		api.PUT("/updateTeacherInformation", AuthWall(), UpdateTeacherInformation)
		api.GET("/getActiveApplications", AuthWall(), GetActiveApplications)
		api.GET("/getAllApplications", AuthWall(), GetAllApplications)
		api.GET("/getNews", AuthWall(), GetNews)
		api.GET("/getAdminApplications", AuthWall(), GetAdminApplications)
		api.GET("/getApplication", AuthWall(), GetApplication)
		api.POST("/createApplication", AuthWall(), CreateApplication)
		api.PUT("/updateApplication", AuthWall(), UpdateApplication)
		api.DELETE("/deleteApplication", AuthWall(), DeleteApplication)
		api.GET("/getAbsenceFormForClasses", AuthWall(), GetAbsenceFormForClasses)
		api.GET("/getAbsenceFormForTeacher", AuthWall(), GetAbsenceFormForTeacher)
		api.GET("/getCompensationForEducationalSupportForm", AuthWall(), GetCompensationForEducationalSupportForm)
		api.GET("/getTravelInvoiceForm", AuthWall(), GetTravelInvoiceForm)
		api.GET("/getBusinessTripApplicationForm", AuthWall(), GetBusinessTripApplicationForm)
		api.GET("/getTravelInvoiceExcel", AuthWall(), GetTravelInvoiceExcel)
		api.GET("/getBusinessTripApplicationExcel", AuthWall(), GetBusinessTripApplicationExcel)
		api.POST("/saveBillingReceipt", AuthWall(), SaveBillingReceipt)
	}

	// Not Found Route
	router.NoRoute(func(context *gin.Context) {
		context.JSON(http.StatusNotFound, Error{"this endpoint doesn't exist"})
	})

	// Providing API
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusMovedPermanently, "swagger/index.html")
	})

	// Starting
	log.Fatal(router.Run(":" + strconv.Itoa(Port)))
}

// setDebugMode analyzes whether a .debug File is present (DebugFilePath)
// if so return true if not false
func debugMode() bool {
	if _, err := os.Stat(DebugFilePath); err == nil {
		return true
	}
	return false
}
