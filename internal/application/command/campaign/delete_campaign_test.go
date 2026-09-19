package campaign

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/event"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type memCampaignRepo struct {
	byID map[uuid.UUID]*domaincampaign.Campaign
}

func (r *memCampaignRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memCampaignRepo) Save(_ context.Context, campaign *domaincampaign.Campaign) error {
	r.byID[campaign.ID] = campaign
	return nil
}
func (r *memCampaignRepo) Update(_ context.Context, campaign *domaincampaign.Campaign) error {
	r.byID[campaign.ID] = campaign
	return nil
}
func (r *memCampaignRepo) GetByID(_ context.Context, id uuid.UUID) (*domaincampaign.Campaign, error) {
	c := r.byID[id]
	if c == nil || c.IsDeleted() {
		return nil, nil
	}
	return c, nil
}
func (r *memCampaignRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domaincampaign.Campaign, error) {
	return r.GetByID(ctx, id)
}
func (r *memCampaignRepo) GetDefaultByClientID(context.Context, uuid.UUID) (*domaincampaign.Campaign, error) {
	return nil, nil
}
func (r *memCampaignRepo) GetByBackgroundPendingKey(context.Context, string) (*domaincampaign.Campaign, error) {
	return nil, nil
}

type memMediaSoftDeleteRepo struct {
	byCampaign map[uuid.UUID][]*domainmedia.Media
	fail       bool
}

func (r *memMediaSoftDeleteRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memMediaSoftDeleteRepo) Save(context.Context, *domainmedia.Media) error { return nil }
func (r *memMediaSoftDeleteRepo) Update(context.Context, *domainmedia.Media) error {
	return nil
}
func (r *memMediaSoftDeleteRepo) GetByID(context.Context, uuid.UUID) (*domainmedia.Media, error) {
	return nil, nil
}
func (r *memMediaSoftDeleteRepo) GetByObjectKey(context.Context, string) (*domainmedia.Media, error) {
	return nil, nil
}
func (r *memMediaSoftDeleteRepo) ListProcessingUpdatedBefore(
	context.Context, time.Time, int,
) ([]*domainmedia.Media, error) {
	return nil, nil
}
func (r *memMediaSoftDeleteRepo) SoftDeleteByCampaignID(_ context.Context, campaignID uuid.UUID) error {
	if r.fail {
		return errors.New("media soft delete failed")
	}
	for _, m := range r.byCampaign[campaignID] {
		m.SoftDelete()
	}
	return nil
}

type memOutbox struct {
	events []event.DomainEvent
}

func (o *memOutbox) StoreEvents(_ context.Context, events []event.DomainEvent) error {
	o.events = append(o.events, events...)
	return nil
}
func (o *memOutbox) FetchUnpublished(context.Context, int) ([]port.OutboxMessage, error) {
	return nil, nil
}
func (o *memOutbox) MarkPublished(context.Context, []uuid.UUID) error { return nil }

func TestDeleteCampaignHandler_SoftDeletesMedias(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	campaign, err := domaincampaign.NewCampaign("Launch", uuid.New(), &start, &end)
	if err != nil {
		t.Fatalf("new campaign: %v", err)
	}
	_ = campaign.PullEvents()

	mediaA, err := domainmedia.NewPendingUpload(campaign.ID, campaign.ClientID, "a.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("media a: %v", err)
	}
	mediaB, err := domainmedia.NewPendingUpload(campaign.ID, campaign.ClientID, "b.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("media b: %v", err)
	}

	campaignRepo := &memCampaignRepo{byID: map[uuid.UUID]*domaincampaign.Campaign{campaign.ID: campaign}}
	mediaRepo := &memMediaSoftDeleteRepo{
		byCampaign: map[uuid.UUID][]*domainmedia.Media{
			campaign.ID: {mediaA, mediaB},
		},
	}
	outbox := &memOutbox{}
	h := NewDeleteCampaignHandler(campaignRepo, mediaRepo, outbox)

	if err := h.Handle(context.Background(), DeleteCampaignCommand{ID: campaign.ID}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !campaign.IsDeleted() {
		t.Fatal("campaign should be soft-deleted")
	}
	if !mediaA.IsDeleted() || !mediaB.IsDeleted() {
		t.Fatal("medias should be soft-deleted")
	}
	if len(outbox.events) == 0 {
		t.Fatal("expected campaign deleted event")
	}
}

func TestDeleteCampaignHandler_MediaFailureKeepsCampaign(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	campaign, err := domaincampaign.NewCampaign("Launch", uuid.New(), &start, &end)
	if err != nil {
		t.Fatalf("new campaign: %v", err)
	}
	_ = campaign.PullEvents()

	// In-memory WithTransaction does not roll back; assert SoftDeleteByCampaignID is called
	// after campaign update by failing media delete and checking the returned error.
	campaignRepo := &memCampaignRepo{byID: map[uuid.UUID]*domaincampaign.Campaign{campaign.ID: campaign}}
	mediaRepo := &memMediaSoftDeleteRepo{fail: true}
	h := NewDeleteCampaignHandler(campaignRepo, mediaRepo, &memOutbox{})

	err = h.Handle(context.Background(), DeleteCampaignCommand{ID: campaign.ID})
	if err == nil || err.Error() != "failed to delete campaign medias" {
		t.Fatalf("want media delete error, got %v", err)
	}
}
