package api

import (
	"fmt"
	"net/http"
	"sigmamono/internal/core"
	"sigmamono/internal/core/access/resource"
	"sigmamono/internal/enum/event"
	"sigmamono/internal/param"
	"sigmamono/internal/response"
	"sigmamono/internal/term"
	"sigmamono/internal/types"
	"sigmamono/model"
	"sigmamono/service"
	"sigmamono/utils/excel"

	"github.com/gin-gonic/gin"
)

const thisUser = "user"
const thisUsers = "users"

// UserAPI for injecting user service
type UserAPI struct {
	Service service.UserServ
	Engine  *core.Engine
}

// ProvideUserAPI for user is used in wire
func ProvideUserAPI(c service.UserServ) UserAPI {
	return UserAPI{Service: c, Engine: c.Engine}
}

// FindByID is used for fetch a user by it's id
func (p *UserAPI) FindByID(c *gin.Context) {
	resp := response.New(p.Engine, c)
	var err error
	var user model.User

	if resp.CheckAccess(resource.CompanyRead) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	if user.ID, err = types.StrToRowID(c.Param("userID")); err != nil {
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	if user, err = p.Service.FindByID(user.ID); err != nil {
		resp.Status(http.StatusNotFound).Error(err).MessageT(term.Record_Not_Found).JSON()
		return
	}

	user.Password = ""

	resp.Record(event.CompanyView)
	resp.Status(http.StatusOK).
		MessageT(term.V_info, thisUser).
		JSON(user)
}

// FindByUsername is used when we try to find a user with username
func (p *UserAPI) FindByUsername(c *gin.Context) {
	resp := response.New(p.Engine, c)
	username := c.Param("username")

	user, err := p.Service.FindByUsername(username)
	if err != nil {
		resp.Status(http.StatusNotFound).Error(err).JSON()
		return
	}

	user.Password = ""

	resp.Status(http.StatusOK).JSON(user)
}

// List of users
func (p *UserAPI) List(c *gin.Context) {
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.NodeWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	params := param.Get(c, p.Engine, thisUsers)

	data, err := p.Service.List(params)
	p.Engine.Debug(err)
	if err != nil {
		resp.Error(err).JSON()
		return
	}

	resp.Record(event.NodeList)
	resp.Status(http.StatusOK).
		MessageT(term.List_of_V, thisUsers).
		JSON(data)
}

// Create user
func (p *UserAPI) Create(c *gin.Context) {
	var user model.User
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.UserWrite) {
		resp.Error(term.You_dont_have_permission).JSON()
		return
	}

	if err := c.ShouldBindJSON(&user); err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, err)
		return
	}

	params := param.Get(c, p.Engine, thisUsers)
	createdUser, err := p.Service.Create(user, params)
	if err != nil {
		resp.Error(err).JSON()
		return
	}

	user.Password = ""
	resp.Record(event.UserCreate, nil, user)

	resp.Status(http.StatusOK).
		Message(term.User_created_successfully).
		JSON(createdUser)
}

// Update user
func (p *UserAPI) Update(c *gin.Context) {
	resp := response.New(p.Engine, c)
	var err error

	if resp.CheckAccess(resource.UserWrite) {
		resp.Error(term.You_dont_have_permission).JSON()
		return
	}

	var user, userBefore, userUpdated model.User

	if user.ID, err = types.StrToRowID(c.Param("userID")); err != nil {
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	if err = c.ShouldBindJSON(&user); err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, err)
		return
	}

	if userBefore, err = p.Service.FindByID(user.ID); err != nil {
		resp.Status(http.StatusNotFound).Error(term.Record_Not_Found).JSON()
		return
	}

	if userUpdated, err = p.Service.Save(user); err != nil {
		resp.Error(err).JSON()
		return
	}

	resp.Record(event.UserUpdate, userBefore, userUpdated)

	resp.Status(http.StatusOK).
		MessageT(term.V_updated_successfully, thisUser).
		JSON(userUpdated)

}

// Delete user
func (p *UserAPI) Delete(c *gin.Context) {
	resp := response.New(p.Engine, c)
	var err error
	var user model.User

	if resp.CheckAccess(resource.UserWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	if user.ID, err = types.StrToRowID(c.Param("userID")); err != nil {
		p.Engine.CheckError(err, err.Error())
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	params := param.Get(c, p.Engine, thisRoles)

	if user, err = p.Service.Delete(user.ID, params); err != nil {
		resp.Status(http.StatusInternalServerError).Error(err).JSON()
		return
	}

	resp.Record(event.UserDelete, user)
	resp.Status(http.StatusOK).
		MessageT(term.V_deleted_successfully, thisUser).
		JSON()
}

// Excel generate excel files based on search
func (p *UserAPI) Excel(c *gin.Context) {
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.UserExcel) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	params := param.Get(c, p.Engine, thisUsers)
	users, err := p.Service.Excel(params)
	if err != nil {
		resp.Status(http.StatusNotFound).Error(term.Record_Not_Found).JSON()
		return
	}

	ex := excel.New("node").
		AddSheet("Nodes").
		AddSheet("Summary").
		Active("Nodes").
		SetPageLayout("landscape", "A4").
		SetPageMargins(0.2).
		SetHeaderFooter().
		SetColWidth("A", "A", 20).
		SetColWidth("B", "C", 15.3).
		SetColWidth("F", "F", 20).
		SetColWidth("L", "M", 20).
		Active("Summary").
		Active("Nodes").
		WriteHeader("ID", "Name", "Username", "Code", "Status", "Role",
			"Language", "Type", "Email", "Readonly", "Direction", "Created At",
			"Updated At")

	for i, v := range users {
		extra := v.Extra.(map[string]interface{})
		column := &[]interface{}{
			v.ID,
			v.Account.Name,
			v.Username,
			v.Account.Code,
			v.Account.Status,
			extra["role"],
			v.Language,
			v.Account.Type,
			v.Email,
			v.Account.Readonly,
			v.Account.Direction,
			v.Account.CreatedAt.Format("2006-01-02 15:04:05"),
			v.Account.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		err = ex.File.SetSheetRow(ex.ActiveSheet, fmt.Sprint("A", i+2), column)
		p.Engine.CheckError(err, "Error in writing to the excel in user")
	}

	ex.Sheets[ex.ActiveSheet].Row = len(users) + 1

	ex.AddTable()

	buffer, downloadName, err := ex.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, &response.Result{
			Message: "Error in generating Excel file",
		})
		return
	}

	resp.Record(event.NodeExcel)

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+downloadName)
	c.Data(http.StatusOK, "application/octet-stream", buffer.Bytes())

}
