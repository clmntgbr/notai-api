package sightengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	domaincontent "go-api/internal/domain/content"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

const DetectorName = "sightengine"

type Config struct {
	APIURL    string
	APIUser   string
	APISecret string
}

type Detector struct {
	cfg        Config
	httpClient *http.Client
	storage    port.Storage
}

func New(cfg Config, storage port.Storage) *Detector {
	return &Detector{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		storage: storage,
	}
}

func (d *Detector) Name() string { return DetectorName }

func (d *Detector) ExpectedWeight() float64 { return 1 }

func (d *Detector) Analyze(
	ctx context.Context,
	contentID uuid.UUID,
	objectKey string,
) ([]domaincontent.Signal, error) {
	_ = contentID
	tmpPath, cleanup, err := d.downloadToTemp(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("sightengine: download: %w", err)
	}
	defer cleanup()

	return d.callAPI(ctx, tmpPath)
}

func (d *Detector) downloadToTemp(ctx context.Context, objectKey string) (string, func(), error) {
	obj, err := d.storage.Get(ctx, objectKey)
	if err != nil {
		return "", nil, err
	}
	defer obj.Close()

	tmpFile, err := os.CreateTemp("", "sightengine-*")
	if err != nil {
		return "", nil, err
	}

	if _, err := io.Copy(tmpFile, obj); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, err
	}

	path := tmpFile.Name()
	cleanup := func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("failed to remove temp file path=%s err=%v", path, err)
		}
	}
	return path, cleanup, nil
}

func (d *Detector) callAPI(ctx context.Context, filePath string) ([]domaincontent.Signal, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	_ = writer.WriteField("models", "genai")
	_ = writer.WriteField("api_user", d.cfg.APIUser)
	_ = writer.WriteField("api_secret", d.cfg.APISecret)
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.cfg.APIURL, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sightengine: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("sightengine: status %d: %s", resp.StatusCode, string(body))
	}

	var out response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("sightengine: decode: %w", err)
	}
	if out.Status != "success" {
		return nil, fmt.Errorf("sightengine: api status %q", out.Status)
	}

	return mapToSignals(out), nil
}

type response struct {
	Status string `json:"status"`
	Type   struct {
		AIGenerated  float64            `json:"ai_generated"`
		AIGenerators map[string]float64 `json:"ai_generators"`
	} `json:"type"`
}

func mapToSignals(out response) []domaincontent.Signal {
	signals := []domaincontent.Signal{
		{
			Type:        "model",
			Code:        "sightengine_genai",
			Description: fmt.Sprintf("Sightengine (GenAI): %.0f%% AI confidence", out.Type.AIGenerated*100),
			Weight:      out.Type.AIGenerated,
		},
	}

	if out.Type.AIGenerated > 0.3 {
		top := topGenerator(out.Type.AIGenerators)
		if top.score > 0.1 {
			signals = append(signals, domaincontent.Signal{
				Type:        "model",
				Code:        "sightengine_generator_match",
				Description: fmt.Sprintf("Probable match: %s (%.0f%%)", top.name, top.score*100),
				Weight:      top.score,
			})
		}
	}

	return signals
}

type generatorScore struct {
	name  string
	score float64
}

func topGenerator(generators map[string]float64) generatorScore {
	var best generatorScore
	for name, score := range generators {
		if score > best.score {
			best = generatorScore{name: name, score: score}
		}
	}
	return best
}

// CleanupTempResiduals removes leftover sightengine-* temp files from a previous crash.
func CleanupTempResiduals() {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "sightengine-*"))
	if err != nil {
		return
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("failed to clean residual temp file path=%s err=%v", path, err)
		}
	}
}
