package service

import (
	"context"

	"campaigntrackerpro/platform/httpx"
	"campaigntrackerpro/services/campaigns"
	"campaigntrackerpro/services/creators"
)

var (
	ErrNotFound   = httpx.ErrNotFound
	ErrValidation = httpx.ErrValidation
)

// Analytics reads across the other services to produce the overview,
// benchmarks, regions and search. It owns no tables of its own — every
// number it reports is derived from what campaigns and creators publish.
//
// It depends on the narrowest interfaces that satisfy it rather than on
// those services directly, so neither can drift into it.
type CampaignReader interface {
	All(ctx context.Context) ([]campaigns.Campaign, error)
	Enrich(ctx context.Context, all []campaigns.Campaign) map[string]campaigns.CampaignRow
	List(ctx context.Context, p campaigns.ListParams) (campaigns.ListResult, error)
	PendingCount(ctx context.Context) (int64, error)
	PendingSpend(ctx context.Context) (int64, error)
	Flagged(ctx context.Context) ([]campaigns.Campaign, error)
	Festivals(ctx context.Context) ([]campaigns.FestivalStat, error)
}

// CreatorReader is only used to make creators findable in the palette.
type CreatorReader interface {
	All(ctx context.Context) ([]creators.Creator, error)
}

type Analytics struct {
	campaigns CampaignReader
	creators  CreatorReader
}

func New(c CampaignReader, cr CreatorReader) *Analytics {
	return &Analytics{campaigns: c, creators: cr}
}
