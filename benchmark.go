package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang/geo/s2"
	"github.com/uber/h3-go/v4"
)

var H3ResolutionAveragesKm2 = map[int]float64{
	0: 4357449.416078381,
	1: 609788.441794133,
	2: 86801.780398997,
	3: 12393.434655088,
	4: 1770.347654491,
	5: 252.903858182,
	6: 36.129062164,
	7: 5.161293360,
	8: 0.737327598,
}

var S2ResolutionAveragesKm2 = map[int]float64{
	0:  85011012.19,
	1:  21252753.05,
	2:  5313188.26,
	3:  1328297.07,
	4:  332074.27,
	5:  83018.57,
	6:  20754.64,
	7:  5188.66,
	8:  1297.17,
	9:  324.29,
	10: 81.07,
	11: 20.27,
	12: 5.07,
	13: 1.27,
}

type Measurement struct {
	Resolution        int
	AverageAreaKm2    float64
	AverageDurationNs float64
}

// GeoJSONFeature represents a single GeoJSON Feature
type GeoJSONFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoJSONGeometry        `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

// GeoJSONGeometry represents the geometry portion of a GeoJSON Feature
type GeoJSONGeometry struct {
	Type        string         `json:"type"`
	Coordinates [][][2]float64 `json:"coordinates"`
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
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("error unmarshaling GeoJSON: %w", err)
	}

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
	// [[exterior ring], [hole1], [hole2], ...]
	// First ring is the exterior boundary, rest are holes

	// Convert exterior ring (first ring)
	exterior := convertRingToGeoLoop(geometry.Coordinates[0])

	// Convert holes (if any)
	var holes []h3.GeoLoop
	if len(geometry.Coordinates) > 1 {
		for holeIndex, holeRing := range geometry.Coordinates[1:] {
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
func ProcessPolygonsWithH3(h3Polygons []h3.GeoPolygon, resolution int, printStuff bool) []time.Duration {
	var durations []time.Duration
	for i, polygon := range h3Polygons {
		if printStuff {
			fmt.Printf("\nProcessing Polygon %d\n", i)
		}

		// Example: Polygon to cells (covering the polygon with H3 cells)
		start := time.Now()
		cells, err := h3.PolygonToCells(polygon, resolution)
		duration := time.Since(start)
		durations = append(durations, duration)
		if err != nil {
			log.Printf("Error converting polygon %d to cells: %v", i, err)
			continue
		}

		if printStuff {
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
	}
	return durations
}

// FeatureRegions holds a Feature and its corresponding S2 regions
type FeatureRegions struct {
	FeatureID int
	Regions   []s2.Region
}

// ConvertGeoJSONToS2Regions reads a GeoJSON file and converts all polygon features to S2 regions
func ConvertGeoJSONToS2Regions(filePath string) ([]FeatureRegions, error) {
	// Read the GeoJSON file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse the GeoJSON FeatureCollection
	var fc GeoJSONFeatureCollection
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("error unmarshaling GeoJSON: %w", err)
	}

	var featureRegions []FeatureRegions

	// Convert each feature to S2 regions
	for i, feature := range fc.Features {
		if feature.Geometry.Type != "Polygon" {
			log.Printf("Warning: Feature %d is not a Polygon, skipping", i)
			continue
		}

		// Get feature ID from properties if available
		featureID := i + 1
		if id, ok := feature.Properties["id"]; ok {
			if idVal, ok := id.(float64); ok {
				featureID = int(idVal)
			}
		}

		// Convert the GeoJSON polygon to S2 regions
		regions, err := convertGeometryToS2Regions(feature.Geometry)
		if err != nil {
			log.Printf("Warning: Error converting feature %d: %v", i, err)
			continue
		}

		featureRegions = append(featureRegions, FeatureRegions{
			FeatureID: featureID,
			Regions:   regions,
		})
	}

	return featureRegions, nil
}

// convertGeometryToS2Regions converts a GeoJSON polygon geometry to S2 regions
func convertGeometryToS2Regions(geometry GeoJSONGeometry) ([]s2.Region, error) {
	if geometry.Type != "Polygon" {
		return nil, fmt.Errorf("expected Polygon geometry, got %s", geometry.Type)
	}

	if len(geometry.Coordinates) == 0 {
		return nil, fmt.Errorf("polygon has no coordinates")
	}

	// Convert exterior ring to S2 loop
	exteriorLoop := convertRingToS2Loop(geometry.Coordinates[0])
	if exteriorLoop == nil {
		return nil, fmt.Errorf("failed to create exterior loop")
	}

	// Build list of all loops (exterior + holes)
	loops := []*s2.Loop{exteriorLoop}

	// Add holes as additional loops
	if len(geometry.Coordinates) > 1 {
		for holeIndex, holeRing := range geometry.Coordinates[1:] {
			if len(holeRing) < 4 {
				log.Printf("Warning: Hole %d has fewer than 4 points, skipping", holeIndex)
				continue
			}

			holeLoop := convertRingToS2Loop(holeRing)
			if holeLoop == nil {
				log.Printf("Warning: Failed to create hole loop %d", holeIndex)
				continue
			}

			loops = append(loops, holeLoop)
		}
	}

	// Create polygon from all loops
	polygon := s2.PolygonFromLoops(loops)

	var regions []s2.Region
	regions = append(regions, polygon)

	return regions, nil
}

// convertRingToS2Loop converts a GeoJSON ring to an S2 Loop
func convertRingToS2Loop(ring [][2]float64) *s2.Loop {
	if len(ring) < 4 {
		return nil
	}

	// Convert each [lon, lat] pair to S2 Point
	// GeoJSON uses [longitude, latitude] order
	points := make([]s2.Point, 0, len(ring)-1) // -1 because first and last are the same

	for i := 0; i < len(ring)-1; i++ { // Skip the last point (duplicate of first)
		coord := ring[i]
		latLng := s2.LatLngFromDegrees(coord[1], coord[0])
		point := s2.PointFromLatLng(latLng)
		points = append(points, point)
	}

	// Create and return the loop
	loop := s2.LoopFromPoints(points)

	return loop
}

func saveFloat64ToCSV(filename string, data map[int]Measurement) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"Resolution", "AvgAreaKm2", "AverageDurationNs"}
	if err := writer.Write(headers); err != nil {
		panic(err)
	}

	// Write each float as a row
	for k, v := range data {
		row := []string{
			strconv.Itoa(k),
			strconv.FormatFloat(v.AverageAreaKm2, 'f', -1, 64),
			strconv.FormatFloat(v.AverageDurationNs, 'f', -1, 64),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return writer.Error()
}

func saveAllTokens(coverings []s2.CellUnion, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, covering := range coverings {
		for _, cellID := range covering {
			_, err := writer.WriteString(cellID.ToToken() + "\n")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessS2Regions demonstrates how to use the S2 regions
func ProcessS2Regions(featureRegions []FeatureRegions,
	minLevel int, maxLevel int, maxCells int, levelMod int, printStuff bool) []time.Duration {
	// Configure RegionCoverer
	rc := &s2.RegionCoverer{
		MinLevel: minLevel,
		MaxLevel: maxLevel,
		MaxCells: maxCells,
		LevelMod: levelMod,
	}
	var grouped []s2.CellUnion
	var durations []time.Duration

	for _, fr := range featureRegions {
		if printStuff {
			fmt.Printf("\nFeature %d has %d region(s)\n", fr.FeatureID, len(fr.Regions))
		}

		for _, region := range fr.Regions {
			// Get covering
			levelCounts := make(map[int]int)
			start := time.Now()
			covering := rc.Covering(region)
			duration := time.Since(start)
			durations = append(durations, duration)
			grouped = append(grouped, covering)

			for _, cell := range covering {
				level := cell.Level()
				levelCounts[level]++
			}

			if printStuff {
				fmt.Printf("\nDuration: %v", duration.Nanoseconds())
				for level, levelCount := range levelCounts {
					fmt.Printf("\nLevel: %d; Level Count: %d\n", level, levelCount)
				}
			}
		}
	}
	return durations
}

func saveToCSV(filename string, header string, data []time.Duration) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{header}); err != nil {
		return err
	}

	// Write data rows
	for _, value := range data {
		row := []string{strconv.FormatInt(value.Nanoseconds(), 10)}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return writer.Error()
}

func durationsToInt64(durations []time.Duration) []int64 {
	ns := make([]int64, len(durations)) // preallocate slice
	for i, d := range durations {
		ns[i] = d.Nanoseconds()
	}
	return ns
}

func averageInt64(nums []int64) float64 {
	if len(nums) == 0 {
		return 0 // avoid division by zero
	}

	var sum int64
	for _, v := range nums {
		sum += v
	}

	return float64(sum) / float64(len(nums))
}

func h3Experiments(filePath string) {

	// H3
	h3Polygons, err := ConvertGeoJSONToH3Polygons(filePath)
	if err != nil {
		log.Fatalf("Error converting GeoJSON to H3 polygons: %v", err)
	}
	fmt.Printf("Successfully converted %d polygons to H3 GeoPolygon format\n", len(h3Polygons))

	fmt.Printf("H3 Experiments ================================================\n")
	maxResolution := 8 // H3 resolution (0-15, higher = smaller cells)
	h3averages := make(map[int]Measurement)
	for i := 0; i <= maxResolution; i++ {
		fmt.Printf("\nResolution: %d\n", i)

		// Inputs
		resolution := i
		output := fmt.Sprintf("/home/nick898/repos/earth-discretization-benchmark/output/durations-h3-res%d.csv", i)
		print := false

		// Test interections
		durations := ProcessPolygonsWithH3(h3Polygons, resolution, print)

		// Save results
		saveToCSV(output, "duration (ns)", durations)
		h3avg := averageInt64(durationsToInt64(durations))
		fmt.Printf("\nAverage: %v\n", h3avg)
		h3averages[i] = Measurement{
			Resolution:        i,
			AverageAreaKm2:    H3ResolutionAveragesKm2[i],
			AverageDurationNs: h3avg,
		}
	}
	saveFloat64ToCSV("/home/nick898/repos/earth-discretization-benchmark/output/h3-averages.csv", h3averages)
}

func s2Experiments(filePath string) {

	featureRegions, err := ConvertGeoJSONToS2Regions(filePath)
	if err != nil {
		log.Fatalf("Error converting GeoJSON to S2 regions: %v", err)
	}
	fmt.Printf("Successfully converted %d features to S2 regions\n", len(featureRegions))

	fmt.Printf("\nS2 Experiments ================================================\n")
	// Fix the max cells and set minLevel = maxLevel and vary the levels
	maxResolution := 13 // Levels 0 - 30; level 13 has average area of 1.27 km^2
	s2averages := make(map[int]Measurement)
	for i := 0; i <= maxResolution; i++ {
		fmt.Printf("\nLevel: %d\n", i)

		// Inputs
		output := fmt.Sprintf("/home/nick898/repos/earth-discretization-benchmark/output/durations-s2-res%d.csv", i)
		minLevel := i
		maxLevel := i
		maxCells := 8 // Default value used; gives a reasonable tradeoff between the number of cells used and the accuracy of the approximation based on source code comments
		levelMod := 1
		print := false

		// Test intersections
		durations := ProcessS2Regions(featureRegions, minLevel, maxLevel, maxCells, levelMod, print)

		// Save results
		saveToCSV(output, "duration (ns)", durations)
		s2avg := averageInt64(durationsToInt64(durations))
		fmt.Printf("\nAverage: %v\n", s2avg)
		s2averages[i] = Measurement{
			Resolution:        i,
			AverageAreaKm2:    S2ResolutionAveragesKm2[i],
			AverageDurationNs: s2avg,
		}
	}
	saveFloat64ToCSV("/home/nick898/repos/earth-discretization-benchmark/output/s2-averages.csv", s2averages)

	// // Fix the level and vary max cells
	// maxCells := 1000
	// s2averages := make([]float64, 0, maxCells)
	// for i := 1; i <= maxCells; i = i + 50 {
	// 	fmt.Printf("\nMax Cells: %d\n", i)

	// 	// Inputs
	// 	// output := fmt.Sprintf("/home/nick898/repos/earth-discretization-benchmark/output/durations-s2-cells-%d.csv", i)
	// 	minLevel := 5
	// 	maxLevel := 13
	// 	levelMod := 1
	// 	print := false

	// 	// Test intersections
	// 	durations := ProcessS2Regions(featureRegions, minLevel, maxLevel, maxCells, levelMod, print)

	// 	// Save results
	// 	// saveToCSV(output, "duration (ns)", durations)
	// 	s2avg := averageInt64(durationsToInt64(durations))
	// 	fmt.Printf("\nAverage: %v\n", s2avg)
	// 	s2averages = append(s2averages, s2avg)
	// }
	// saveFloat64ToCSV("/home/nick898/repos/earth-discretization-benchmark/output/s2-averages-maxcells.csv", "avg_duration_ns", s2averages)

}

func main() {
	filePath := "/home/nick898/repos/earth-discretization-benchmark/data/mock_polygons.geojson"
	h3Experiments(filePath)
	s2Experiments(filePath)
}
