package content

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	domaincampaign "go-api/internal/domain/campaign"
	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/event"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const presignExpiry = 15 * time.Minute

type PresignFileInput struct {
	Filename    string
	ContentType string
}

type PresignContentsCommand struct {
	CampaignID uuid.UUID
	ClientID   uuid.UUID
	Files      []PresignFileInput
}

type PresignContentItem struct {
	ContentID uuid.UUID
	URL       string
	ObjectKey string
	Filename  string
}

type PresignContentsResult struct {
	CampaignID uuid.UUID
	Items      []PresignContentItem
}

type PresignContentsHandler struct {
	campaignRepo domaincampaign.CampaignWriteRepository
	contentRepo  domaincontent.ContentWriteRepository
	outbox       port.OutboxRepository
	storage      port.Storage
}

func NewPresignContentsHandler(
	campaignRepo domaincampaign.CampaignWriteRepository,
	contentRepo domaincontent.ContentWriteRepository,
	outbox port.OutboxRepository,
	storage port.Storage,
) *PresignContentsHandler {
	return &PresignContentsHandler{
		campaignRepo: campaignRepo,
		contentRepo:  contentRepo,
		outbox:       outbox,
		storage:      storage,
	}
}

func (h *PresignContentsHandler) Handle(
	ctx context.Context,
	cmd PresignContentsCommand,
) (*PresignContentsResult, error) {
	if len(cmd.Files) == 0 {
		return nil, domaincontent.ErrEmptyFileList
	}
	if len(cmd.Files) > domaincontent.MaxPresignBatch {
		return nil, domaincontent.ErrTooManyFiles
	}

	normalized := make([]PresignFileInput, 0, len(cmd.Files))
	for _, file := range cmd.Files {
		filename := filepath.Base(strings.TrimSpace(file.Filename))
		if err := domaincontent.ValidateContentFilename(filename); err != nil {
			return nil, err
		}
		normalized = append(normalized, PresignFileInput{
			Filename:    filename,
			ContentType: strings.TrimSpace(file.ContentType),
		})
	}

	created := make([]*domaincontent.Content, 0, len(normalized))

	err := h.contentRepo.WithTransaction(ctx, func(txCtx context.Context) error {
		campaign, err := h.resolveCampaign(txCtx, cmd.CampaignID, cmd.ClientID)
		if err != nil {
			return err
		}

		var events []event.DomainEvent
		for _, file := range normalized {
			item, err := domaincontent.NewPendingUpload(
				campaign.ID,
				cmd.ClientID,
				file.Filename,
				file.ContentType,
			)
			if err != nil {
				return err
			}
			if err := h.contentRepo.Save(txCtx, item); err != nil {
				return errors.New("failed to create content")
			}
			created = append(created, item)
			events = append(events, item.PullEvents()...)
		}
		return h.outbox.StoreEvents(txCtx, events)
	})
	if err != nil {
		if errors.Is(err, domaincontent.ErrUnsupportedContentType) ||
			errors.Is(err, domaincontent.ErrInvalidFilename) {
			return nil, err
		}
		return nil, err
	}

	items := make([]PresignContentItem, 0, len(created))
	for _, item := range created {
		url, err := h.storage.PresignedPutURL(ctx, item.ObjectKey, presignExpiry)
		if err != nil {
			return nil, errors.New("failed to generate upload url")
		}
		items = append(items, PresignContentItem{
			ContentID: item.ID,
			URL:       url,
			ObjectKey: item.ObjectKey,
			Filename:  item.Filename,
		})
	}

	return &PresignContentsResult{
		CampaignID: created[0].CampaignID,
		Items:      items,
	}, nil
}

func (h *PresignContentsHandler) resolveCampaign(
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
