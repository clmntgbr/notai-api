package heuristic

import (
	"context"
	"fmt"

	domaincontent "go-api/internal/domain/content"
)

type ExifCheck struct{}

func (ExifCheck) Name() string { return "exif" }

func (ExifCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx

	if tool, generative, found := hasC2PAMarker(in.RawBytes); found && generative {
		return []domaincontent.Signal{{
			Type:        "metadata",
			Code:        "c2pa_generative",
			Description: fmt.Sprintf("Content Credentials declare generative origin (%s)", tool),
			Weight:      p.WeightFor("c2pa_generative"),
		}}, nil
	}

	switch in.Format {
	case "jpeg", "jpg":
		if !hasJPEGExif(in.RawBytes) {
			return []domaincontent.Signal{{
				Type:        "metadata",
				Code:        "exif_missing",
				Description: "No EXIF metadata found",
				Weight:      p.WeightFor("exif_missing"),
			}}, nil
		}
		return []domaincontent.Signal{
			okSignal("exif", "EXIF metadata present; no generative Content Credentials"),
		}, nil
	default:
		return []domaincontent.Signal{
			skippedSignal("exif", fmt.Sprintf(
				"EXIF presence not scored for %s; no generative Content Credentials",
				in.Format,
			)),
		}, nil
	}
}
