// path: controllers/reports.go
package controllers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"meniba/database"
	"meniba/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JSON payload for POST /api/reports when Content-Type: application/json
type ReportJSON struct {
	Category       string  `json:"category"`
	Note           string  `json:"note"`
	AreaLabel      string  `json:"area_label"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	AccuracyM      *int    `json:"accuracy_m,omitempty"`
	PrivacyRadiusM *int    `json:"privacy_radius_m,omitempty"`
	Anonymous      bool    `json:"anonymous"`
        Adrehs    string `json:"adrehs,omitempty"`
	District  string `json:"district,omitempty"`
	Chiefdom  string `json:"chiefdom,omitempty"`
	Region    string `json:"region,omitempty"`
	Section   string `json:"section,omitempty"`
	GeoMethod string `json:"geo_method,omitempty"`

}

func HandlePostReport(c *fiber.Ctx) error {
	ct := c.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		return handleReportJSON(c)
	}
	if strings.HasPrefix(ct, "multipart/form-data") {
		return handleReportMultipart(c)
	}
	return c.Status(fiber.StatusUnsupportedMediaType).
		JSON(ErrorResp{OK: false, Error: "unsupported content type"})
}

func handleReportJSON(c *fiber.Ctx) error {
	var p ReportJSON
	if err := c.BodyParser(&p); err != nil {
		return badReq(c, "invalid JSON")
	}
	if err := validateReport(p.Category, p.Note, p.AreaLabel, p.Lat, p.Lng); err != nil {
		return badReq(c, err.Error())
	}
	if p.PrivacyRadiusM == nil {
		def := 300
		p.PrivacyRadiusM = &def
	}

	doc := models.Report{
		Category:       p.Category,
		Note:           p.Note,
		AreaLabel:      p.AreaLabel,
		Lat:            p.Lat,
		Lng:            p.Lng,
		AccuracyM:      p.AccuracyM,
		PrivacyRadiusM: p.PrivacyRadiusM,
		Anonymous:      p.Anonymous,
                Adrehs:         strings.TrimSpace(p.Adrehs),
		District:       strings.TrimSpace(p.District),
		Chiefdom:       strings.TrimSpace(p.Chiefdom),
		Region:         strings.TrimSpace(p.Region),
		Section:        strings.TrimSpace(p.Section),
		GeoMethod:      strings.TrimSpace(p.GeoMethod),
		CreatedAt:      time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
	defer cancel()
	res, err := database.Col("reports").InsertOne(ctx, doc)
	if err != nil {
		return serverErr(c, err)
	}
	id := res.InsertedID.(primitive.ObjectID).Hex()
	return c.Status(fiber.StatusOK).JSON(models.CreateReportResp{OK: true, ID: id})
}

func handleReportMultipart(c *fiber.Ctx) error {
	// fields
	category := strings.TrimSpace(c.FormValue("category"))
	note := strings.TrimSpace(c.FormValue("note"))
	areaLabel := strings.TrimSpace(c.FormValue("area_label"))
	latStr := c.FormValue("lat")
	lngStr := c.FormValue("lng")
	accStr := c.FormValue("accuracy_m")
	radiusStr := c.FormValue("privacy_radius_m")
	anonStr := c.FormValue("anonymous")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return badReq(c, "invalid lat")
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return badReq(c, "invalid lng")
	}
	var acc *int
	if accStr != "" {
		if v, e := strconv.Atoi(accStr); e == nil {
			acc = &v
		} else {
			return badReq(c, "invalid accuracy_m")
		}
	}
	radius := 300
	if radiusStr != "" {
		if v, e := strconv.Atoi(radiusStr); e == nil {
			radius = v
		} else {
			return badReq(c, "invalid privacy_radius_m")
		}
	}
	anonymous := parseBool(anonStr)

	if err := validateReport(category, note, areaLabel, lat, lng); err != nil {
		return badReq(c, err.Error())
	}

	// files
	uploadDir := getenv("UPLOAD_DIR", "uploads")
	var voiceSaved string
	var photoPaths []string

	// multiple files per key supported
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for key, files := range form.File {
			if len(files) == 0 {
				continue
			}
			switch {
			case key == "voice":
				if voiceSaved == "" {
					if p, e := saveFormFile(uploadDir, "voice", files[0]); e == nil {
						voiceSaved = p
					} else {
						return serverErr(c, e)
					}
				}
			case strings.HasPrefix(key, "photo"):
				for _, fh := range files {
					if p, e := saveFormFile(uploadDir, "photo", fh); e == nil {
						photoPaths = append(photoPaths, p)
					} else {
						return serverErr(c, e)
					}
				}
			}
		}
	} else {
		// fallback single
		if f, err := c.FormFile("voice"); err == nil && f != nil {
			if p, e := saveFormFile(uploadDir, "voice", f); e == nil {
				voiceSaved = p
			} else {
				return serverErr(c, e)
			}
		}
		if f, err := c.FormFile("photo1"); err == nil && f != nil {
			if p, e := saveFormFile(uploadDir, "photo", f); e == nil {
				photoPaths = append(photoPaths, p)
			} else {
				return serverErr(c, e)
			}
		}
	}

	doc := models.Report{
		Category:       category,
		Note:           note,
		AreaLabel:      areaLabel,
		Lat:            lat,
		Lng:            lng,
		AccuracyM:      acc,
		PrivacyRadiusM: &radius,
		Anonymous:      anonymous,
		VoiceURL:       voiceSaved,
		PhotoURLs:      photoPaths,
                Adrehs:    strings.TrimSpace(c.FormValue("adrehs")),
		District:  strings.TrimSpace(c.FormValue("district")),
		Chiefdom:  strings.TrimSpace(c.FormValue("chiefdom")),
		Region:    strings.TrimSpace(c.FormValue("region")),
		Section:   strings.TrimSpace(c.FormValue("section")),
		GeoMethod: strings.TrimSpace(c.FormValue("geo_method")),
		CreatedAt: time.Now().UTC(),		
	}

	ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
	defer cancel()
	res, err := database.Col("reports").InsertOne(ctx, doc)
	if err != nil {
		return serverErr(c, err)
	}
	id := res.InsertedID.(primitive.ObjectID).Hex()
	return c.Status(fiber.StatusOK).JSON(models.CreateReportResp{OK: true, ID: id})
}

func validateReport(category, note, area string, lat, lng float64) error {
	if category == "" {
		return errors.New("missing category")
	}
	if strings.TrimSpace(note) == "" {
		return errors.New("missing note")
	}
	if strings.TrimSpace(area) == "" {
		return errors.New("missing area_label")
	}
	if lat == 0 && lng == 0 {
		return errors.New("invalid coordinates")
	}
	return nil
}

func saveFormFile(uploadDir, prefix string, f *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(f.Filename))
	if len(ext) > 8 {
		ext = ext[:8]
	}
	name := fmt.Sprintf("%s_%d_%s%s", prefix, time.Now().UnixNano(), randString(6), ext)
	dst := filepath.Join(uploadDir, name)
	if err := cpyFile(f, dst); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}

func cpyFile(fh *multipart.FileHeader, dst string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
