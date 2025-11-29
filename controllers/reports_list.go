// path: controllers/reports_list.go
package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"meniba/database"
	"meniba/models"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReportItem struct {
	ID             string   `json:"id"`
	Category       string   `json:"category"`
	Note           string   `json:"note"`
	AreaLabel      string   `json:"area_label"`
	Lat            float64  `json:"lat"`
	Lng            float64  `json:"lng"`
	AccuracyM      *int     `json:"accuracy_m,omitempty"`
	PrivacyRadiusM *int     `json:"privacy_radius_m,omitempty"`
	Anonymous      bool     `json:"anonymous"`
	CreatedAt      string   `json:"created_at"`
	VoiceURL       string   `json:"voice_url,omitempty"`
	PhotoURLs      []string `json:"photo_urls,omitempty"`
}

type ReportListResp struct {
	OK         bool         `json:"ok"`
	Items      []ReportItem `json:"items"`
	NextCursor string       `json:"next_cursor,omitempty"`
}

func HandleListReports(c *fiber.Ctx) error {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 1 {
				n = 1
			}
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}

	filter := bson.M{}

	if cat := c.Query("category"); cat != "" {
		filter["category"] = cat
	}
	if sd := c.Query("start_date"); sd != "" {
		if t, err := time.Parse(time.RFC3339, sd); err == nil {
			setRange(filter, "created_at", "$gte", t)
		} else {
			return badReq(c, "invalid start_date (RFC3339)")
		}
	}
	if ed := c.Query("end_date"); ed != "" {
		if t, err := time.Parse(time.RFC3339, ed); err == nil {
			setRange(filter, "created_at", "$lte", t)
		} else {
			return badReq(c, "invalid end_date (RFC3339)")
		}
	}

	if hm := c.Query("has_media"); hm != "" {
		has := parseBool(hm)
		if has {
			filter["$or"] = []bson.M{
				{"voice_url": bson.M{"$exists": true, "$ne": ""}},
				{"photo_urls.0": bson.M{"$exists": true}},
			}
		} else {
			filter["$and"] = append(asArr(filter["$and"]),
				bson.M{"$or": []bson.M{
					{"voice_url": bson.M{"$exists": false}},
					{"voice_url": ""},
				}},
				bson.M{"photo_urls.0": bson.M{"$exists": false}},
			)
		}
	}

	if bb := c.Query("bbox"); bb != "" {
		minLng, minLat, maxLng, maxLat, err := parseBbox(bb)
		if err != nil {
			return badReq(c, "invalid bbox (minLng,minLat,maxLng,maxLat)")
		}
		filter["lat"] = bson.M{"$gte": minLat, "$lte": maxLat}
		filter["lng"] = bson.M{"$gte": minLng, "$lte": maxLng}
	}

	if cursorHex := c.Query("cursor"); cursorHex != "" {
		if oid, err := primitive.ObjectIDFromHex(cursorHex); err == nil {
			filter["_id"] = bson.M{"$lt": oid}
		} else {
			return badReq(c, "invalid cursor")
		}
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(int64(limit + 1))

	ctx, cancel := context.WithTimeout(c.Context(), 8*time.Second)
	defer cancel()

	cur, err := database.Col("reports").Find(ctx, filter, findOpts)
	if err != nil {
		return serverErr(c, err)
	}
	defer cur.Close(ctx)

	items := make([]ReportItem, 0, limit)
	var nextCursor string
	count := 0

	for cur.Next(ctx) {
		var doc models.Report
		if err := cur.Decode(&doc); err != nil {
			return serverErr(c, err)
		}
		count++
		if count > limit {
			nextCursor = doc.ID.Hex()
			break
		}
		items = append(items, ReportItem{
			ID:             doc.ID.Hex(),
			Category:       doc.Category,
			Note:           doc.Note,
			AreaLabel:      doc.AreaLabel,
			Lat:            doc.Lat,
			Lng:            doc.Lng,
			AccuracyM:      doc.AccuracyM,
			PrivacyRadiusM: doc.PrivacyRadiusM,
			Anonymous:      doc.Anonymous,
			CreatedAt:      doc.CreatedAt.UTC().Format(time.RFC3339),
			VoiceURL:       doc.VoiceURL,
			PhotoURLs:      doc.PhotoURLs,
		})
	}
	if err := cur.Err(); err != nil {
		return serverErr(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(ReportListResp{
		OK:         true,
		Items:      items,
		NextCursor: nextCursor,
	})
}

// helpers reused here

func setRange(m bson.M, key, op string, t time.Time) {
	if m[key] == nil {
		m[key] = bson.M{}
	}
	m[key].(bson.M)[op] = t
}

func parseBbox(s string) (minLng, minLat, maxLng, maxLat float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("need 4 numbers")
	}
	parse := func(i int) (float64, error) { return strconv.ParseFloat(strings.TrimSpace(parts[i]), 64) }
	if minLng, err = parse(0); err != nil {
		return
	}
	if minLat, err = parse(1); err != nil {
		return
	}
	if maxLng, err = parse(2); err != nil {
		return
	}
	if maxLat, err = parse(3); err != nil {
		return
	}
	if maxLng < minLng || maxLat < minLat {
		return 0, 0, 0, 0, fmt.Errorf("max must be >= min")
	}
	return
}

func asArr(v any) []bson.M {
	if v == nil {
		return []bson.M{}
	}
	if a, ok := v.([]bson.M); ok {
		return a
	}
	return []bson.M{}
}
