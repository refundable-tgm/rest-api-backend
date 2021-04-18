package rest

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

const Port = 8080

func StartService() {
	InitTokenManager()
	router := gin.Default()
	//gin.SetMode(gin.ReleaseMode)
	router.POST("/login", Login)
	router.POST("/logout", AuthWall(), Logout)
	router.POST("/login/refresh", Refresh)
	router.GET("/getTeacherByShort", AuthWall(), GetTeacherByShort)
	router.GET("/getTeacher", AuthWall(), GetTeacher)
	router.GET("/setTeacherPermissions", AuthWall(), SetTeacherPermissions)
	router.GET("/getActiveApplications", AuthWall(), GetActiveApplications)
	router.GET("/getAllApplications", AuthWall(), GetAllApplication)
	router.GET("/getNews", AuthWall(), GetNews)
	router.GET("/getAdminApplication", AuthWall(), GetAdminApplication)
	router.GET("/getApplication", AuthWall(), GetApplication)
	router.POST("/createApplication", AuthWall(), CreateApplication)
	router.PUT("/updateApplication", AuthWall(), UpdateApplication)
	router.DELETE("/deleteApplication", AuthWall(), DeleteApplication)
	router.GET("/getAbsenceFormForClasses", AuthWall(), GetAbsenceFormForClasses)
	router.GET("/getAbsenceFormForTeacher", AuthWall(), GetAbsenceFormForTeacher)
	router.GET("/getCompensationForEducationalSupportForm", AuthWall(), GetCompensationForEducationalSupportForm)
	router.GET("/getTravelInvoiceForm", AuthWall(), GetTravelInvoiceForm)
	router.GET("/getBusinessTripApplicationForm", AuthWall(), GetBusinessTripApplicationForm)
	router.GET("/getTravelInvoiceExcel", AuthWall(), GetTravelInvoiceExcel)
	router.GET("/getBusinessTripApplicationExcel", AuthWall(), GetBusinessTripApplicationExcel)
	router.POST("/saveBillingReceipt", AuthWall(), SaveBillingReceipt)

	router.NoRoute(func(context *gin.Context) {
		context.JSON(http.StatusNotFound, fmt.Errorf("this endpoint doesn't exist"))
	})

	log.Fatal(router.Run(":" + strconv.Itoa(Port)))
}
