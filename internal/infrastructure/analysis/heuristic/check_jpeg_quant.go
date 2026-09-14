package heuristic

import (
	"context"
	"fmt"

	domaincontent "go-api/internal/domain/content"
)

type JPEGQuantCheck struct{}

func (JPEGQuantCheck) Name() string { return "jpeg_quant" }

func (JPEGQuantCheck) Run(ctx context.Context, in CheckInput, p CheckParams) ([]domaincontent.Signal, error) {
	_ = ctx
	if in.Format != "jpeg" && in.Format != "jpg" {
		return []domaincontent.Signal{
			skippedSignal("jpeg_quant", fmt.Sprintf(
				"JPEG quantization check not applicable for %s",
				in.Format,
			)),
		}, nil
	}
	tables, err := parseDQTMarkers(in.RawBytes)
	if err != nil {
		return nil, err
	}
	tool, ok := isGenericExportQuant(tables)
	if !ok {
		return []domaincontent.Signal{
			okSignal("jpeg_quant", "Quantization tables do not match known generic export profiles"),
		}, nil
	}
	return []domaincontent.Signal{{
		Type: "heuristic",
		Code: "jpeg_quant_generic",
		Description: fmt.Sprintf(
			"Generic quantization table (%s), no real camera signature",
			tool,
		),
		Weight: p.WeightFor("jpeg_quant_generic"),
	}}, nil
}
