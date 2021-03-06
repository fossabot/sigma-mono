package param

import (
	"sigmamono/internal/core"
	"sigmamono/internal/term"
	"sigmamono/internal/types"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Get is a function for filling param.Model
func Get(c *gin.Context, engine *core.Engine, part string) (param Param) {
	var err error

	generateOrder(c, &param, part)
	generateSelectedColumns(c, &param)
	generateLimit(c, &param, engine)
	generateOffset(c, &param, engine)

	param.Search = strings.TrimSpace(c.Query("search"))

	userID, ok := c.Get("USER_ID")
	if ok {
		engine.CheckInfo(err, "User ID is not exist")
		param.UserID = userID.(types.RowID)
	}

	language, ok := c.Get("LANGUAGE")
	if ok {
		param.Language = language.(string)
	}

	companyIDTmp, ok := c.Get("COMPANY_ID")
	if !ok {
		engine.CheckInfo(err, term.CompanyID_not_exist_in_context)
	}

	nodeCodeTmp, ok := c.Get("NODE_CODE")
	if !ok {
		engine.CheckInfo(err, term.NodeCode_not_exist_in_context)
	}

	param.CompanyID = companyIDTmp.(types.RowID)
	param.NodeCode = nodeCodeTmp.(uint64)

	return param
}

func generateOrder(c *gin.Context, param *Param, part string) {
	orderBy := part + ".id"
	direction := "desc"

	if c.Query("order_by") != "" {
		orderBy = c.Query("order_by")
	}

	if c.Query("direction") != "" {
		direction = c.Query("direction")
	}

	param.Order = orderBy + " " + direction
}

func generateSelectedColumns(c *gin.Context, param *Param) {
	param.Select = "*"
	if c.Query("select") != "" {
		param.Select = c.Query("select")
	}
}

func generateLimit(c *gin.Context, param *Param, engine *core.Engine) {
	var err error
	param.Limit = 10
	if c.Query("page_size") != "" {
		param.Limit, err = strconv.ParseUint(c.Query("page_size"), 10, 16)
		if err != nil {
			// TODO: get path from gin.Context
			engine.CheckError(err, "Limit is not number")
			param.Limit = 10
		}
	}
}

func generateOffset(c *gin.Context, param *Param, engine *core.Engine) {
	var page uint64
	page = 0
	var err error
	if c.Query("page") != "" {
		page, err = strconv.ParseUint(c.Query("page"), 10, 16)
		if err != nil {
			// TODO: get path from gin.Context
			engine.CheckError(err, "Offset is not a positive number")
			page = 0
		}
	}

	param.Offset = param.Limit * (page)
}
