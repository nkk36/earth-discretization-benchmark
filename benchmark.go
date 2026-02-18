package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/uber/h3-go/v4"
)

// GeoJSONFeature represents a single GeoJSON Feature
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// GeoJSONGeometry represents the geometry portion of a GeoJSON Feature
type GeoJSONGeometry struct {
	Type        string           `json:"type"`
	Coordinates [][][][2]float64 `json:"coordinates"`
}

// GeoJSONFeatureCollection represents a GeoJSON FeatureCollection
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// ConvertGeoJSONToH3Polygons reads a GeoJSON file and converts all polygons to H3 GeoPolygons
func ConvertGeoJSONToH3Polygons(filePath string) ([]h3.GeoPolygon, error) {
	// Read the GeoJSON file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse the GeoJSON FeatureCollection
	var fc GeoJSONFeatureCollection
	log.Printf("Before unmarshal")
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("error unmarshaling GeoJSON: %w", err)
	}
	log.Printf("After unmarshal")

	var h3Polygons []h3.GeoPolygon

	// Convert each feature to an H3 GeoPolygon
	for i, feature := range fc.Features {
		if feature.Geometry.Type != "Polygon" {
			log.Printf("Warning: Feature %d is not a Polygon, skipping", i)
			continue
		}

		// Convert the GeoJSON polygon to H3 GeoPolygon
		h3Polygon, err := convertGeometryToH3Polygon(feature.Geometry)
		if err != nil {
			log.Printf("Warning: Error converting feature %d: %v", i, err)
			continue
		}

		h3Polygons = append(h3Polygons, h3Polygon)
	}

	return h3Polygons, nil
}

// convertGeometryToH3Polygon converts a GeoJSON geometry to an H3 GeoPolygon
func convertGeometryToH3Polygon(geometry GeoJSONGeometry) (h3.GeoPolygon, error) {
	if geometry.Type != "Polygon" {
		return h3.GeoPolygon{}, fmt.Errorf("expected Polygon geometry, got %s", geometry.Type)
	}

	if len(geometry.Coordinates) == 0 {
		return h3.GeoPolygon{}, fmt.Errorf("polygon has no coordinates")
	}

	// GeoJSON Polygon coordinates are organized as:
	// [[[exterior ring], [hole1], [hole2], ...]]
	// First ring is the exterior boundary, rest are holes

	// Convert exterior ring (first ring)
	if len(geometry.Coordinates[0]) < 1 {
		return h3.GeoPolygon{}, fmt.Errorf("polygon has no exterior ring")
	}

	exterior := convertRingToGeoLoop(geometry.Coordinates[0][0])

	// Convert holes (if any)
	var holes []h3.GeoLoop
	if len(geometry.Coordinates[0]) > 1 {
		for holeIndex, holeRing := range geometry.Coordinates[0][1:] {
			if len(holeRing) < 4 {
				log.Printf("Warning: Hole %d has fewer than 4 points, skipping", holeIndex)
				continue
			}
			hole := convertRingToGeoLoop(holeRing)
			holes = append(holes, hole)
		}
	}

	// Create H3 GeoPolygon
	h3Polygon := h3.GeoPolygon{
		GeoLoop: exterior,
		Holes:   holes,
	}

	return h3Polygon, nil
}

// convertRingToGeoLoop converts a GeoJSON ring to an H3 GeoLoop
func convertRingToGeoLoop(ring [][2]float64) h3.GeoLoop {
	var geoLoop h3.GeoLoop

	// Convert each [lon, lat] pair to H3 LatLng
	// GeoJSON uses [longitude, latitude] order
	for _, coord := range ring {
		geoLoc := h3.LatLng{
			Lat: coord[1], // latitude
			Lng: coord[0], // longitude
		}
		geoLoop = append(geoLoop, geoLoc)
	}

	return geoLoop
}

// ProcessPolygonsWithH3 is an example function showing how to use the H3 polygons
func ProcessPolygonsWithH3(h3Polygons []h3.GeoPolygon, resolution int) error {
	for i, polygon := range h3Polygons {
		fmt.Printf("\nProcessing Polygon %d\n", i)

		// Example: Polygon to cells (covering the polygon with H3 cells)
		cells, err := h3.PolygonToCells(polygon, resolution)
		if err != nil {
			log.Printf("Error converting polygon %d to cells: %v", i, err)
			continue
		}

		fmt.Printf("Polygon %d covers %d H3 cells at resolution %d\n", i, len(cells), resolution)

		// Example: Get some cell information
		if len(cells) > 0 {
			fmt.Printf("  First few cell IDs: ")
			for j := 0; j < 3 && j < len(cells); j++ {
				fmt.Printf("%v ", cells[j])
			}
			fmt.Println()
		}
	}

	return nil
}

func main() {
	// Example usage
	filePath := "/home/nick898/repos/earth-discretization-benchmark/data/mock_polygons.geojson"
	resolution := 9 // H3 resolution (0-15, higher = smaller cells)

	// Read GeoJSON and convert to H3 polygons
	h3Polygons, err := ConvertGeoJSONToH3Polygons(filePath)
	if err != nil {
		log.Fatalf("Error converting GeoJSON to H3 polygons: %v", err)
	}

	fmt.Printf("Successfully converted %d polygons to H3 GeoPolygon format\n", len(h3Polygons))

	// Process the polygons with H3
	if err := ProcessPolygonsWithH3(h3Polygons, resolution); err != nil {
		log.Fatalf("Error processing polygons: %v", err)
	}
}
