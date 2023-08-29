package controllers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/geekgonecrazy/rfd-tool/config"
	"github.com/geekgonecrazy/rfd-tool/utils"
	"github.com/gin-gonic/gin"
)

func handleError(c *gin.Context, verboseMsg string, reportedError error) {
	id, err := utils.NewUUID()
	if err != nil {
		log.Println("Error Generating Error Code", err)
	}

	log.Println(fmt.Sprintf("Error: %s Verbose: %s Error: ", id, verboseMsg), reportedError)

	c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{"siteName": config.Config.Site.Name, "requestId": id})
}

func handleErrorJSON(c *gin.Context, verboseMsg string, err error) {
	id, err2 := utils.NewUUID()
	if err2 != nil {
		log.Println("Error Generating Error Code", err2)
	}

	log.Println(fmt.Sprintf("Error: %s Verbose: %s Error: ", id, verboseMsg), err)

	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "requestId": id})
}

// LivenessCheckHandler liveness check
func LivenessCheckHandler(c *gin.Context) {
	c.AbortWithStatus(http.StatusOK)
}
