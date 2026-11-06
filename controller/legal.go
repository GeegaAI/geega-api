package controller

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// loadLegalDocsRow 读取当前生效内容（version 最大的行）。
// 查询失败（如未迁移的只读从库没有该表）时按空内容放行，避免阻断业务。
func loadLegalDocsRow() *model.LegalDocVersion {
	row, err := model.GetCurrentLegalDocs()
	if err != nil {
		common.SysLog("failed to load legal docs, treating as empty: " + err.Error())
		return &model.LegalDocVersion{}
	}
	return row
}

func legalDocsConfigured(row *model.LegalDocVersion) bool {
	return system_setting.IsLegalValueConfigured(row.UserAgreement) ||
		system_setting.IsLegalValueConfigured(row.PrivacyPolicy) ||
		system_setting.IsLegalValueConfigured(row.ServiceAgreement)
}

// GetLegalDocs 返回当前生效的三份文档内容与名称（root，管理端编辑用）。
func GetLegalDocs(c *gin.Context) {
	row := loadLegalDocsRow()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    row,
	})
}

type saveLegalDocsRequest struct {
	UserAgreement        string `json:"user_agreement"`
	PrivacyPolicy        string `json:"privacy_policy"`
	ServiceAgreement     string `json:"service_agreement"`
	UserAgreementName    string `json:"user_agreement_name"`
	PrivacyPolicyName    string `json:"privacy_policy_name"`
	ServiceAgreementName string `json:"service_agreement_name"`
}

// SaveLegalDocs 保存编辑内容：就地更新当前版本行（不升版本，用户端立即生效）。
func SaveLegalDocs(c *gin.Context) {
	var req saveLegalDocsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	saved, err := model.SaveCurrentLegalDocs(&model.LegalDocVersion{
		UserAgreement:        req.UserAgreement,
		PrivacyPolicy:        req.PrivacyPolicy,
		ServiceAgreement:     req.ServiceAgreement,
		UserAgreementName:    req.UserAgreementName,
		PrivacyPolicyName:    req.PrivacyPolicyName,
		ServiceAgreementName: req.ServiceAgreementName,
	}, c.GetInt("id"))
	if err != nil {
		common.SysLog("SaveLegalDocs failed: " + err.Error())
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"version": saved.Version,
		},
	})
}

type legalVersionItem struct {
	Version           int    `json:"version"`
	PublishedAt       int64  `json:"published_at"`
	PublishedBy       int    `json:"published_by"`
	PublisherUsername string `json:"publisher_username"`
}

// GetLegalDocVersions 返回当前版本号与最近版本历史（root）。
func GetLegalDocVersions(c *gin.Context) {
	current, err := model.GetCurrentLegalDocVersion()
	if err != nil {
		common.SysLog("GetLegalDocVersions failed to load current version: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}
	versions, err := model.GetLegalDocVersions(20)
	if err != nil {
		common.SysLog("GetLegalDocVersions failed to load versions: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgDatabaseError)
		return
	}

	items := make([]legalVersionItem, 0, len(versions))
	userIds := make([]int, 0, len(versions))
	for _, version := range versions {
		items = append(items, legalVersionItem{
			Version:     version.Version,
			PublishedAt: version.PublishedAt,
			PublishedBy: version.PublishedBy,
		})
		userIds = append(userIds, version.PublishedBy)
	}
	usernames := make(map[int]string, len(userIds))
	if len(userIds) > 0 {
		var users []model.User
		if err := model.DB.Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err == nil {
			for _, user := range users {
				usernames[user.Id] = user.Username
			}
		}
	}
	for index := range items {
		items[index].PublisherUsername = usernames[items[index].PublishedBy]
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"current_version": current,
			"versions":        items,
		},
	})
}

// PublishLegalVersion 把当前内容克隆为新版本，触发所有用户重新勾选确认（root）。
func PublishLegalVersion(c *gin.Context) {
	version, err := model.PublishLegalDocs(c.GetInt("id"))
	if err != nil {
		if errors.Is(err, model.ErrLegalDocsEmpty) {
			common.ApiErrorMsg(c, "Legal documents are not configured yet")
			return
		}
		common.SysLog("PublishLegalVersion failed: " + err.Error())
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"version": version.Version,
		},
	})
}
