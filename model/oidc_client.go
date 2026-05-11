package model

import (
	"errors"

	"gorm.io/gorm"
)

var ErrOIDCClientNotFound = errors.New("oidc client not found or disabled")

// OIDCClient represents a registered OIDC Relying Party (e.g., lobehub).
type OIDCClient struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	ClientId     string `json:"client_id" gorm:"uniqueIndex;type:varchar(128);not null"`
	ClientSecret string `json:"client_secret" gorm:"type:varchar(128);not null"`
	RedirectURI  string `json:"redirect_uri" gorm:"type:varchar(512);not null"`
	Name         string `json:"name" gorm:"type:varchar(255);not null"`
	Enabled      bool   `json:"enabled" gorm:"default:true"`
}

func (c *OIDCClient) TableName() string {
	return "oidc_clients"
}

func GetOIDCClients() ([]*OIDCClient, error) {
	var clients []*OIDCClient
	err := DB.Where("enabled = ?", true).Find(&clients).Error
	return clients, err
}

func GetOIDCClientByClientId(clientId string) (*OIDCClient, error) {
	if clientId == "" {
		return nil, errors.New("client_id is empty")
	}
	var client OIDCClient
	err := DB.Where("client_id = ? AND enabled = ?", clientId, true).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOIDCClientNotFound
		}
		return nil, err
	}
	return &client, nil
}

func (c *OIDCClient) Insert() error {
	return DB.Create(c).Error
}

func (c *OIDCClient) Update() error {
	return DB.Save(c).Error
}

func (c *OIDCClient) Delete() error {
	return DB.Delete(c).Error
}

// AutoMigrateOIDCClients creates the oidc_clients table if it doesn't exist
func AutoMigrateOIDCClients() error {
	return DB.AutoMigrate(&OIDCClient{})
}
