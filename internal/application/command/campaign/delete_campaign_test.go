package campaign

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
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

type memMediaDeleteRepo struct {
	byID       map[uuid.UUID]*domainmedia.Media
	updateFail bool
}

func (r *memMediaDeleteRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memMediaDeleteRepo) Save(_ context.Context, media *domainmedia.Media) error {
	r.byID[media.ID] = media
	return nil
}
func (r *memMediaDeleteRepo) Update(_ context.Context, media *domainmedia.Media) error {
	if r.updateFail {
		return errors.New("update failed")
	}
	r.byID[media.ID] = media
	return nil
}
func (r *memMediaDeleteRepo) GetByID(_ context.Context, id uuid.UUID) (*domainmedia.Media, error) {
	m := r.byID[id]
	if m == nil || m.IsDeleted() {
		return nil, nil
	}
	return m, nil
}
func (r *memMediaDeleteRepo) GetByObjectKey(context.Context, string) (*domainmedia.Media, error) {
	return nil, nil
}
func (r *memMediaDeleteRepo) ListProcessingUpdatedBefore(
	context.Context, time.Time, int,
) ([]*domainmedia.Media, error) {
	return nil, nil
}
func (r *memMediaDeleteRepo) ListActiveByCampaignID(
	_ context.Context,
	campaignID uuid.UUID,
) ([]*domainmedia.Media, error) {
	out := make([]*domainmedia.Media, 0)
	for _, m := range r.byID {
		if m.CampaignID == campaignID && !m.IsDeleted() {
			out = append(out, m)
		}
	}
	return out, nil
}

type memContentDeleteRepo struct {
	byMedia map[uuid.UUID][]domaincontent.Content
}

func (r *memContentDeleteRepo) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (r *memContentDeleteRepo) Save(context.Context, *domaincontent.Content) error { return nil }
func (r *memContentDeleteRepo) Update(context.Context, *domaincontent.Content) error {
	return nil
}
func (r *memContentDeleteRepo) GetByID(context.Context, uuid.UUID) (*domaincontent.Content, error) {
	return nil, nil
}
func (r *memContentDeleteRepo) GetByObjectKey(context.Context, string) (*domaincontent.Content, error) {
	return nil, nil
}
func (r *memContentDeleteRepo) ListByMediaID(_ context.Context, mediaID uuid.UUID) ([]domaincontent.Content, error) {
	return r.byMedia[mediaID], nil
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

func TestDeleteCampaignHandler_SoftDeletesMediasAndEmitsMediaDeleted(t *testing.T) {
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
	_ = mediaA.PullEvents()
	mediaB, err := domainmedia.NewPendingUpload(campaign.ID, campaign.ClientID, "b.jpg", "image/jpeg")
	if err != nil {
		t.Fatalf("media b: %v", err)
	}
	_ = mediaB.PullEvents()

	thumb := "thumb-a.jpg"
	contentA := domaincontent.Content{
		ID:           uuid.New(),
		MediaID:      mediaA.ID,
		ObjectKey:    mediaA.ObjectKey,
		ThumbnailKey: &thumb,
	}

	campaignRepo := &memCampaignRepo{byID: map[uuid.UUID]*domaincampaign.Campaign{campaign.ID: campaign}}
	mediaRepo := &memMediaDeleteRepo{byID: map[uuid.UUID]*domainmedia.Media{
		mediaA.ID: mediaA,
		mediaB.ID: mediaB,
	}}
	contentRepo := &memContentDeleteRepo{byMedia: map[uuid.UUID][]domaincontent.Content{
		mediaA.ID: {contentA},
	}}
	outbox := &memOutbox{}
	h := NewDeleteCampaignHandler(campaignRepo, mediaRepo, contentRepo, outbox)

	if err := h.Handle(context.Background(), DeleteCampaignCommand{ID: campaign.ID}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !campaign.IsDeleted() {
		t.Fatal("campaign should be soft-deleted")
	}
	if !mediaA.IsDeleted() || !mediaB.IsDeleted() {
		t.Fatal("medias should be soft-deleted")
	}

	var mediaDeleted int
	for _, e := range outbox.events {
		if e.EventType() == domainmedia.EventTypeMediaDeleted {
			mediaDeleted++
			deleted, ok := e.(domainmedia.MediaDeleted)
			if !ok {
				t.Fatalf("unexpected event type %T", e)
			}
			if deleted.MediaID == mediaA.ID.String() {
				if deleted.ObjectKey != mediaA.ObjectKey {
					t.Fatalf("object key: got %s", deleted.ObjectKey)
				}
				if len(deleted.Contents) != 1 || deleted.Contents[0].ThumbnailKey != thumb {
					t.Fatalf("contents: %+v", deleted.Contents)
				}
			}
		}
	}
	if mediaDeleted != 2 {
		t.Fatalf("media.deleted events: got %d want 2", mediaDeleted)
	}
}

func TestDeleteCampaignHandler_MediaUpdateFailure(t *testing.T) {
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
	_ = mediaA.PullEvents()

	campaignRepo := &memCampaignRepo{byID: map[uuid.UUID]*domaincampaign.Campaign{campaign.ID: campaign}}
	mediaRepo := &memMediaDeleteRepo{
		byID:       map[uuid.UUID]*domainmedia.Media{mediaA.ID: mediaA},
		updateFail: true,
	}
	h := NewDeleteCampaignHandler(campaignRepo, mediaRepo, &memContentDeleteRepo{}, &memOutbox{})

	err = h.Handle(context.Background(), DeleteCampaignCommand{ID: campaign.ID})
	if err == nil || err.Error() != "failed to delete campaign medias" {
		t.Fatalf("want media delete error, got %v", err)
	}
}
