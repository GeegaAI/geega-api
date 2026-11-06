package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// LegalDocVersion 法律文档内容版本表，同时是协议内容的唯一存储：
// 最新一行（version 最大）即当前生效内容；编辑就地更新最新行（不升版本，
// 用户端立即生效），点“发布新版本”克隆一行 version+1（所有用户需重新勾选确认）。
type LegalDocVersion struct {
	Id                   int    `json:"id" gorm:"primaryKey"`
	Version              int    `json:"version" gorm:"uniqueIndex;not null"`
	UserAgreement        string `json:"user_agreement" gorm:"type:text"`
	PrivacyPolicy        string `json:"privacy_policy" gorm:"type:text"`
	ServiceAgreement     string `json:"service_agreement" gorm:"type:text"`
	UserAgreementName    string `json:"user_agreement_name" gorm:"type:text"`
	PrivacyPolicyName    string `json:"privacy_policy_name" gorm:"type:text"`
	ServiceAgreementName string `json:"service_agreement_name" gorm:"type:text"`
	PublishedBy          int    `json:"published_by"`
	PublishedAt          int64  `json:"published_at"`
}

var ErrLegalDocsEmpty = errors.New("legal docs not initialized")

func GetCurrentLegalDocVersion() (int, error) {
	var current int
	err := DB.Model(&LegalDocVersion{}).
		Select("COALESCE(MAX(version), 0)").
		Scan(&current).Error
	return current, err
}

// GetCurrentLegalDocs 返回当前生效内容（version 最大的行）；无任何版本时返回零值行。
func GetCurrentLegalDocs() (*LegalDocVersion, error) {
	var row LegalDocVersion
	err := DB.Order("version desc").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &LegalDocVersion{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// SaveCurrentLegalDocs 把编辑内容写进最新一行（就地更新，不升版本）；
// 尚无任何版本时创建 version 1。
func SaveCurrentLegalDocs(docs *LegalDocVersion, publishedBy int) (*LegalDocVersion, error) {
	current, err := GetCurrentLegalDocs()
	if err != nil {
		return nil, err
	}
	if current.Version == 0 {
		created := &LegalDocVersion{
			Version:              1,
			UserAgreement:        docs.UserAgreement,
			PrivacyPolicy:        docs.PrivacyPolicy,
			ServiceAgreement:     docs.ServiceAgreement,
			UserAgreementName:    docs.UserAgreementName,
			PrivacyPolicyName:    docs.PrivacyPolicyName,
			ServiceAgreementName: docs.ServiceAgreementName,
			PublishedBy:          publishedBy,
			PublishedAt:          common.GetTimestamp(),
		}
		if err := DB.Create(created).Error; err != nil {
			return nil, err
		}
		return created, nil
	}
	updates := map[string]any{
		"user_agreement":         docs.UserAgreement,
		"privacy_policy":         docs.PrivacyPolicy,
		"service_agreement":      docs.ServiceAgreement,
		"user_agreement_name":    docs.UserAgreementName,
		"privacy_policy_name":    docs.PrivacyPolicyName,
		"service_agreement_name": docs.ServiceAgreementName,
	}
	if err := DB.Model(&LegalDocVersion{}).
		Where("version = ?", current.Version).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	return GetCurrentLegalDocs()
}

// PublishLegalDocs 把当前内容克隆为新版本（version+1），触发所有用户重新勾选确认。
func PublishLegalDocs(publishedBy int) (*LegalDocVersion, error) {
	current, err := GetCurrentLegalDocs()
	if err != nil {
		return nil, err
	}
	if current.Version == 0 {
		return nil, ErrLegalDocsEmpty
	}
	created := &LegalDocVersion{
		Version:              current.Version + 1,
		UserAgreement:        current.UserAgreement,
		PrivacyPolicy:        current.PrivacyPolicy,
		ServiceAgreement:     current.ServiceAgreement,
		UserAgreementName:    current.UserAgreementName,
		PrivacyPolicyName:    current.PrivacyPolicyName,
		ServiceAgreementName: current.ServiceAgreementName,
		PublishedBy:          publishedBy,
		PublishedAt:          common.GetTimestamp(),
	}
	if err := DB.Create(created).Error; err != nil {
		return nil, err
	}
	return created, nil
}

// GetLegalDocVersions 返回最近 limit 个版本，按版本号倒序。
func GetLegalDocVersions(limit int) ([]*LegalDocVersion, error) {
	var versions []*LegalDocVersion
	err := DB.Order("version desc").Limit(limit).Find(&versions).Error
	return versions, err
}

// MigrateLegalDocsFromOptions 一次性数据迁移：版本表为空且旧 option 里存有
// 协议内容时，把 option 值迁入为 version 1。幂等，仅在空表时执行。
func MigrateLegalDocsFromOptions() error {
	var count int64
	if err := DB.Model(&LegalDocVersion{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var options []*Option
	if err := DB.Where("key IN ?", []string{
		"legal.user_agreement",
		"legal.privacy_policy",
		"legal.service_agreement",
		"legal.user_agreement_name",
		"legal.privacy_policy_name",
		"legal.service_agreement_name",
	}).Find(&options).Error; err != nil {
		return err
	}
	values := make(map[string]string, len(options))
	for _, option := range options {
		values[option.Key] = option.Value
	}
	if values["legal.user_agreement"] == "" &&
		values["legal.privacy_policy"] == "" &&
		values["legal.service_agreement"] == "" {
		return nil
	}

	created := &LegalDocVersion{
		Version:              1,
		UserAgreement:        values["legal.user_agreement"],
		PrivacyPolicy:        values["legal.privacy_policy"],
		ServiceAgreement:     values["legal.service_agreement"],
		UserAgreementName:    values["legal.user_agreement_name"],
		PrivacyPolicyName:    values["legal.privacy_policy_name"],
		ServiceAgreementName: values["legal.service_agreement_name"],
		PublishedAt:          common.GetTimestamp(),
	}
	if err := DB.Create(created).Error; err != nil {
		return err
	}
	common.SysLog("legal docs migrated from legacy options into legal_doc_versions v1")
	return nil
}
