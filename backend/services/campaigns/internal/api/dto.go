package api

import (
	"campaigntrackerpro/services/campaigns"
	"time"

	"campaigntrackerpro/services/campaigns/internal/service"
)

const rfc3339 = time.RFC3339

type CampaignDetailDTO struct {
	campaigns.CampaignDTO
	Creatives []campaigns.CreativeDTO   `json:"creatives"`
	Audit     []campaigns.AuditEventDTO `json:"audit"`
	Similar   []campaigns.CampaignDTO   `json:"similar"`
	Position  int                       `json:"position"`
	Total     int                       `json:"total"`
	PrevID    string                    `json:"prevId,omitempty"`
	NextID    string                    `json:"nextId,omitempty"`
	MedAll    float64                   `json:"medAll"`
}

func NewCampaignDetailDTO(d service.CampaignDetail) CampaignDetailDTO {
	return CampaignDetailDTO{
		CampaignDTO: campaigns.NewCampaignDTO(d.CampaignRow),
		Creatives:   campaigns.NewCreativeDTOs(d.Creatives),
		Audit:       campaigns.NewAuditDTOs(d.Audit),
		Similar:     campaigns.NewCampaignDTOs(d.Similar),
		Position:    d.Position,
		Total:       d.Total,
		PrevID:      d.PrevID,
		NextID:      d.NextID,
		MedAll:      d.MedAll,
	}
}
