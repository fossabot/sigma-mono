package api

import (
	"net/http"
	"sigma/domain/accounting/service"
	"sigma/domain/core"
	"sigma/domain/core/access/resource"
	"sigma/internal/enum/event"
	"sigma/internal/param"
	"sigma/internal/response"
	"sigma/internal/term"
	"sigma/internal/types"
	"sigma/model"
	"sigma/utils/excel"

	"github.com/gin-gonic/gin"
)

const thisAccount = "account"
const thisAccounts = "accounts"

// AccountAPI for injecting account service
type AccountAPI struct {
	Service service.AccountServ
	Engine  *core.Engine
}

// ProvideAccountAPI for account is used in wire
func ProvideAccountAPI(c service.AccountServ) AccountAPI {
	return AccountAPI{Service: c, Engine: c.Engine}
}

// FindByID is used for fetch a account by it's id
func (p *AccountAPI) FindByID(c *gin.Context) {
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.AccountRead) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	accountID, err := types.StrToRowID(c.Param("accountID"))
	if err != nil {
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	account, err := p.Service.FindByID(types.RowID(accountID))
	if err != nil {
		resp.Status(http.StatusNotFound).Error(err).MessageT(term.Record_Not_Found).JSON()
		return
	}

	p.Engine.Record(c, event.AccountView)
	resp.Status(http.StatusOK).
		MessageT(term.V_info, thisAccount).
		JSON(account)
}

// List of accounts
func (p *AccountAPI) List(c *gin.Context) {
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.AccountWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	params := param.Get(c, p.Engine, thisAccounts)

	data, err := p.Service.List(params)
	p.Engine.Debug(err)
	if err != nil {
		resp.Error(err).JSON()
		return
	}

	p.Engine.Record(c, event.AccountList)
	resp.Status(http.StatusOK).
		MessageT(term.List_of_V, thisAccounts).
		JSON(data)
}

// Create account
func (p *AccountAPI) Create(c *gin.Context) {
	var account model.Account
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.AccountWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	if err := c.ShouldBindJSON(&account); err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, err)
		return
	}

	params := param.Get(c, p.Engine, thisAccounts)

	createdAccount, err := p.Service.Create(account, params)
	if err != nil {
		resp.Error(err).JSON()
		return
	}

	p.Engine.Record(c, event.AccountCreate, nil, account)

	resp.Status(http.StatusOK).
		MessageT(term.V_created_successfully, thisAccount).
		JSON(createdAccount)
}

// Update account
func (p *AccountAPI) Update(c *gin.Context) {
	resp := response.New(p.Engine, c)
	var err error

	if resp.CheckAccess(resource.AccountWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	var account, accountBefore, accountUpdated model.Account

	account.ID, err = types.StrToRowID(c.Param("accountID"))
	if err != nil {
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	if err = c.ShouldBindJSON(&account); err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, err)
		return
	}

	if accountBefore, err = p.Service.FindByID(account.ID); err != nil {
		resp.Status(http.StatusNotFound).Error(term.Record_Not_Found).JSON()
		return
	}

	if accountUpdated, err = p.Service.Save(account); err != nil {
		resp.Error(err).JSON()
		return
	}

	p.Engine.Record(c, event.AccountUpdate, accountBefore, account)

	resp.Status(http.StatusOK).
		MessageT(term.V_updated_successfully, thisAccount).
		JSON(accountUpdated)
}

// Delete account
func (p *AccountAPI) Delete(c *gin.Context) {
	resp := response.New(p.Engine, c)
	var err error
	var account model.Account

	if resp.CheckAccess(resource.AccountWrite) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	if account.ID, err = types.StrToRowID(c.Param("accountID")); err != nil {
		resp.Error(term.Invalid_ID).JSON()
		return
	}

	if account, err = p.Service.Delete(account.ID); err != nil {
		resp.Status(http.StatusInternalServerError).Error(err).JSON()
		return
	}

	p.Engine.Record(c, event.AccountDelete, account)
	resp.Status(http.StatusOK).
		MessageT(term.V_deleted_successfully, thisAccount).
		JSON()
}

// Excel generate excel files based on search
func (p *AccountAPI) Excel(c *gin.Context) {
	resp := response.New(p.Engine, c)

	if resp.CheckAccess(resource.AccountExcel) {
		resp.Status(http.StatusForbidden).Error(term.You_dont_have_permission).JSON()
		return
	}

	params := param.Get(c, p.Engine, thisAccounts)
	accounts, err := p.Service.Excel(params)
	if err != nil {
		resp.Status(http.StatusNotFound).Error(term.Record_Not_Found).JSON()
		return
	}

	p.Engine.Record(c, event.AccountExcel)

	ex := excel.New("account")
	ex.AddSheet("Accounts").
		AddSheet("Summary").
		Active("Accounts").
		SetPageLayout("landscape", "A4").
		SetPageMargins(0.2).
		SetHeaderFooter().
		SetColWidth("B", "C", 15.3).
		SetColWidth("M", "M", 20).
		Active("Summary").
		SetColWidth("A", "D", 20).
		Active("Accounts").
		WriteHeader("ID", "Name", "Legal Name", "Server Address", "Expiration", "Plan",
			"Detail", "Phone", "Email", "Website", "Type", "Code", "Updated At").
		SetSheetFields("ID", "Name", "LegalName", "ServerAddress", "Expiration", "Plan",
			"Detail", "Phone", "Email", "Website", "Type", "Code", "UpdatedAt").
		WriteData(accounts).
		AddTable()

	buffer, downloadName, err := ex.Generate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, &response.Result{
			Message: "Error in generating Excel file",
		})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+downloadName)
	c.Data(http.StatusOK, "application/octet-stream", buffer.Bytes())

}
