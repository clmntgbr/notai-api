package media

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	"go-api/internal/domain/event"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const presignExpiry = 15 * time.Minute

type PresignFileInput struct {
	Filename    string
	ContentType string
}

type PresignMediaCommand struct {
	CampaignID uuid.UUID
	ClientID   uuid.UUID
	Files      []PresignFileInput
}

type PresignMediaItem struct {
	MediaID   uuid.UUID
	URL       string
	ObjectKey string
	Filename  string
	MediaType domainmedia.MediaType
}

type PresignMediaResult struct {
	CampaignID uuid.UUID
	Items      []PresignMediaItem
}

type PresignMediaHandler struct {
	campaignRepo domaincampaign.CampaignWriteRepository
	mediaRepo    domainmedia.MediaWriteRepository
	outbox       port.OutboxRepository
	storage      port.Storage
}

func NewPresignMediaHandler(
	campaignRepo domaincampaign.CampaignWriteRepository,
	mediaRepo domainmedia.MediaWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
) *PresignMediaHandler {
	return &PresignMediaHandler{
		campaignRepo: campaignRepo,
		mediaRepo:    mediaRepo,
		outbox:       outbox,
		storage:      storage,
	}
}

func (h *PresignMediaHandler) Handle(
	ctx context.Context,
	cmd PresignMediaCommand,
) (*PresignMediaResult, error) {
	if len(cmd.Files) == 0 {
		return nil, domainmedia.ErrEmptyFileList
	}
	if len(cmd.Files) > domainmedia.MaxPresignBatch {
		return nil, domainmedia.ErrTooManyFiles
	}

	normalized := make([]PresignFileInput, 0, len(cmd.Files))
	for _, file := range cmd.Files {
		filename := filepath.Base(strings.TrimSpace(file.Filename))
		if err := domainmedia.ValidateFilename(filename); err != nil {
			return nil, err
		}
		normalized = append(normalized, PresignFileInput{
			Filename:    filename,
			ContentType: strings.TrimSpace(file.ContentType),
		})
	}

	created := make([]*domainmedia.Media, 0, len(normalized))

	err := h.mediaRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.resolveCampaign(txCtx, cmd.CampaignID, cmd.ClientID)
		if err != nil {
			return err
		}

		var events []event.DomainEvent
		for _, file := range normalized {
			item, err := domainmedia.NewPendingUpload(
				campaign.ID,
				cmd.ClientID,
				file.Filename,
				file.ContentType,
			)
			if err != nil {
				return err
			}
			if err := h.mediaRepo.Save(txCtx, item); err != nil {
				return errors.New("failed to create media")
			}
			created = append(created, item)
			events = append(events, item.PullEvents()...)
		}
		return h.outbox.StoreEvents(txCtx, events)
	})
	if err != nil {
		if errors.Is(err, domainmedia.ErrUnsupportedContentType) ||
			errors.Is(err, domainmedia.ErrInvalidFilename) {
			return nil, err
		}
		return nil, err
	}

	items := make([]PresignMediaItem, 0, len(created))
	for _, item := range created {
		url, err := h.storage.PresignedPutURL(ctx, item.ObjectKey, presignExpiry)
		if err != nil {
			return nil, errors.New("failed to generate upload url")
		}
		items = append(items, PresignMediaItem{
			MediaID:   item.ID,
			URL:       url,
			ObjectKey: item.ObjectKey,
			Filename:  item.Filename,
			MediaType: item.MediaType,
		})
	}

	return &PresignMediaResult{
		CampaignID: created[0].CampaignID,
		Items:      items,
	}, nil
}

func (h *PresignMediaHandler) resolveCampaign(
	ctx context.Context,
	campaignID, clientID uuid.UUID,
) (*domaincampaign.Campaign, error) {
	var (
		campaign *domaincampaign.Campaign
		err      error
	)
	if campaignID == uuid.Nil {
		campaign, err = h.campaignRepo.GetDefaultByClientID(ctx, clientID)
	} else {
		campaign, err = h.campaignRepo.GetByID(ctx, campaignID)
	}
	if err != nil {
		return nil, errors.New("failed to get campaign")
	}
	if campaign == nil || campaign.ClientID != clientID {
		return nil, errors.New("campaign not found")
	}
	return campaign, nil
}
